// Package gitrepo generates static pages for browsing git repositories, in the
// style of stagit. Files with a "Repository" metadata property (typically set
// via front matter) are annotated with repository metadata, and log, file
// listing, refs, commit, and file pages are generated alongside them. Generated
// pages are HTML files with a "Layout" property, intended to be rendered by the
// "layout" plugin. Commit diffs and file contents are provided as page content
// (available to templates as "Content") and are cached by git object hash, so
// only new commits and changed files are diffed or read on subsequent builds;
// all pages are still dispatched (and laid out) on every build, so use
// MaxCommits to bound the work for repositories with long histories.
//
// A single branch is tracked per repository: the branch referenced by HEAD,
// unless overridden with a "Branch" metadata property. If HEAD refers to a
// branch with no commits yet, pages are generated with no commits or files, and
// Repository.Head is nil. For example, given "projects/demo/index.md" containing:
//
//	---
//	Repository: demo
//	Layout: repo
//	---
//
// and a chain such as:
//
//	gs.Begin(srcDir).
//		Chain(frontmatter.New()).
//		Chain(gitrepo.New(reposDir)).
//		Chain(markdown.New()).
//		Chain(syntax.New()). // optional: highlights file pages
//		Chain(layout.New()).
//		End(dstDir)
//
// the following pages are generated in "projects/demo":
//
//	log.html             layout "gitlog"
//	files.html           layout "gitfiles"
//	refs.html            layout "gitrefs"
//	commit/<hash>.html   layout "gitcommit", with diffstat and diff as content
//	file/<path>.html     layout "gitfile", with file contents as content
//
// Pages are generated in the directory of the source page, so each directory
// may contain only one source page with a "Repository" property; pages for
// multiple repositories in the same directory overwrite each other.
//
// Every page, including the source page, has a "Repo" property (*Repository)
// and a "RepoRoot" property containing the relative path back to the
// repository directory (e.g. "../" on commit pages), for building links such as
// {{.Props.RepoRoot}}{{.Page}}. Commit pages also have a "Commit" property
// (*Commit) and file pages an "Entry" property (*Entry).
//
// Repository descriptions and clone URLs are read from the "description" and
// "url" files in the git directory, as with stagit.
//
// A top-level markdown README (preferred over other README files) is rendered
// to Repository.ReadmeHTML for display on the source page. Raw HTML in it is
// omitted, as are links with unsafe schemes such as "javascript:". Relative
// links in it that point to files in the tracked branch are rewritten to their
// file pages, and relative images are rewritten to raw copies of the files,
// which are written to "raw/<path>". Links that do not resolve to a file are
// left unchanged.
//
// When using devserver, include the repository directories in the watch list
// so that new commits trigger a rebuild.
package gitrepo

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"html/template"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/chroma/lexers"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	"go.foosoft.net/goldsmith"
)

// Repository contains metadata about a repository and its tracked branch.
type Repository struct {
	Name        string
	Description string
	CloneURL    string
	Branch      string
	Path        string // site-relative directory containing the repository pages

	Head     *Commit   // nil if the tracked branch has no commits yet
	Commits  []*Commit // newest first, limited by MaxCommits
	Branches []*Ref
	Tags     []*Ref // newest first
	Files    []*Entry
	Readme   *Entry
	License  *Entry

	ReadmeHTML template.HTML // rendered markdown README, with links relative to the repository directory
}

// Commit contains metadata about a single commit.
type Commit struct {
	Hash      string
	ShortHash string
	Parents   []string

	Author      string
	AuthorEmail string
	AuthorDate  time.Time

	Subject string
	Message string

	Page string // page URL relative to the repository directory, empty if no page was generated

	object *object.Commit
}

// Ref contains metadata about a branch or tag.
type Ref struct {
	Name   string
	Commit *Commit
}

// Entry contains metadata about a file in the tracked branch.
type Entry struct {
	Path string
	Name string
	Mode string
	Size int64
	Hash string
	Page string // page URL relative to the repository directory
	Raw  string // raw file URL relative to the repository directory, set only for README images

	file *object.File
}

// GitRepo chainable context.
type GitRepo struct {
	rootDir     string
	repoKey     string
	branchKey   string
	layoutKey   string
	maxCommits  int
	maxFileSize int64
}

// New creates a new instance of the GitRepo plugin. Relative repository paths
// are resolved against rootDir.
func New(rootDir string) *GitRepo {
	return &GitRepo{
		rootDir:     rootDir,
		repoKey:     "Repository",
		branchKey:   "Branch",
		layoutKey:   "Layout",
		maxFileSize: 1 << 20,
	}
}

// RepositoryKey sets the metadata key used to access the repository path (default: "Repository").
func (self *GitRepo) RepositoryKey(key string) *GitRepo {
	self.repoKey = key
	return self
}

// BranchKey sets the metadata key used to override the tracked branch (default: "Branch").
func (self *GitRepo) BranchKey(key string) *GitRepo {
	self.branchKey = key
	return self
}

// LayoutKey sets the metadata key used to assign layouts to generated pages (default: "Layout").
func (self *GitRepo) LayoutKey(key string) *GitRepo {
	self.layoutKey = key
	return self
}

// MaxCommits limits the number of commits listed in the log and given commit
// pages; zero or less means no limit (default: 0).
func (self *GitRepo) MaxCommits(count int) *GitRepo {
	self.maxCommits = count
	return self
}

// MaxFileSize sets the largest file size, in bytes, whose contents are
// rendered on file pages; zero or less means no limit (default: 1 MiB).
func (self *GitRepo) MaxFileSize(size int64) *GitRepo {
	self.maxFileSize = size
	return self
}

func (*GitRepo) Name() string {
	return "gitrepo"
}

func (self *GitRepo) Initialize(context *goldsmith.Context) error {
	context.CacheKey(strconv.FormatInt(self.maxFileSize, 10))
	return nil
}

func (self *GitRepo) Process(context *goldsmith.Context, inputFile *goldsmith.File) error {
	repoProp, ok := inputFile.Prop(self.repoKey)
	if !ok {
		context.DispatchFile(inputFile)
		return nil
	}
	repoPath, ok := repoProp.(string)
	if !ok || repoPath == "" {
		return fmt.Errorf("invalid %s property: %v", self.repoKey, repoProp)
	}
	if !filepath.IsAbs(repoPath) {
		repoPath = filepath.Join(self.rootDir, repoPath)
	}

	var branch string
	if branchProp, ok := inputFile.Prop(self.branchKey); ok {
		if branch, ok = branchProp.(string); !ok || branch == "" {
			return fmt.Errorf("invalid %s property: %v", self.branchKey, branchProp)
		}
	}

	gen, err := newGenerator(self, context, inputFile, repoPath, branch)
	if err == nil {
		err = gen.generate()
	}
	if err != nil {
		return fmt.Errorf("repository %s: %w", repoPath, err)
	}

	return nil
}

type generator struct {
	plugin    *GitRepo
	context   *goldsmith.Context
	indexFile *goldsmith.File
	repo      *git.Repository
	head      *object.Commit // nil if the tracked branch has no commits yet
	data      *Repository
	commits   map[plumbing.Hash]*Commit
	files     map[string]*Entry
}

func newGenerator(plugin *GitRepo, context *goldsmith.Context, indexFile *goldsmith.File, repoPath, branch string) (*generator, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, err
	}

	refName := plumbing.NewBranchReferenceName(branch)
	if branch == "" {
		head, err := repo.Reference(plumbing.HEAD, false)
		if err != nil {
			return nil, err
		}
		if refName = head.Target(); !refName.IsBranch() {
			return nil, errors.New("HEAD is not a branch")
		}
	}

	gitDir := repoPath
	if info, err := os.Stat(filepath.Join(repoPath, ".git")); err == nil && info.IsDir() {
		gitDir = filepath.Join(repoPath, ".git")
	}

	name := filepath.Base(filepath.Clean(repoPath))
	if name == ".git" {
		name = filepath.Base(filepath.Dir(filepath.Clean(repoPath)))
	}

	description := readMetaFile(gitDir, "description")
	if strings.HasPrefix(description, "Unnamed repository") {
		description = ""
	}

	gen := &generator{
		plugin:    plugin,
		context:   context,
		indexFile: indexFile,
		repo:      repo,
		commits:   make(map[plumbing.Hash]*Commit),
		files:     make(map[string]*Entry),
		data: &Repository{
			Name:        strings.TrimSuffix(name, ".git"),
			Description: description,
			CloneURL:    readMetaFile(gitDir, "url"),
			Branch:      refName.Short(),
			Path:        indexFile.Dir(),
		},
	}

	ref, err := repo.Reference(refName, true)
	if errors.Is(err, plumbing.ErrReferenceNotFound) && branch == "" {
		return gen, nil // HEAD refers to a branch with no commits yet
	}
	if err != nil {
		return nil, fmt.Errorf("branch %s: %w", refName.Short(), err)
	}
	if gen.head, err = repo.CommitObject(ref.Hash()); err != nil {
		return nil, err
	}
	gen.data.Head = gen.commit(gen.head)

	return gen, nil
}

func readMetaFile(gitDir, name string) string {
	data, err := os.ReadFile(filepath.Join(gitDir, name))
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(data))
}

func (self *generator) commit(commit *object.Commit) *Commit {
	if data, ok := self.commits[commit.Hash]; ok {
		return data
	}

	subject, _, _ := strings.Cut(strings.TrimSpace(commit.Message), "\n")
	data := &Commit{
		Hash:        commit.Hash.String(),
		ShortHash:   commit.Hash.String()[:7],
		Author:      commit.Author.Name,
		AuthorEmail: commit.Author.Email,
		AuthorDate:  commit.Author.When,
		Subject:     strings.TrimSpace(subject),
		Message:     commit.Message,
		object:      commit,
	}
	for _, parent := range commit.ParentHashes {
		data.Parents = append(data.Parents, parent.String())
	}

	self.commits[commit.Hash] = data
	return data
}

func (self *generator) generate() error {
	if err := self.collectRefs(); err != nil {
		return err
	}
	if self.head != nil {
		if err := self.collectLog(); err != nil {
			return err
		}
		if err := self.collectFiles(); err != nil {
			return err
		}
		if err := self.renderReadme(); err != nil {
			return fmt.Errorf("failed to render README: %w", err)
		}
	}

	for _, page := range []struct{ path, layout string }{
		{"log.html", "gitlog"},
		{"files.html", "gitfiles"},
		{"refs.html", "gitrefs"},
	} {
		if err := self.dispatchPage(page.path, page.layout, "", nil, nil); err != nil {
			return err
		}
	}

	for _, commit := range self.data.Commits {
		err := self.dispatchPage(commit.Page, "gitcommit", commit.Hash, goldsmith.FileProps{"Commit": commit}, func() ([]byte, error) {
			return renderDiff(commit.object)
		})
		if err != nil {
			return fmt.Errorf("failed to generate page for commit %s: %w", commit.Hash, err)
		}
	}

	for _, entry := range self.data.Files {
		err := self.dispatchPage("file/"+entry.Path+".html", "gitfile", entry.Hash, goldsmith.FileProps{"Entry": entry}, func() ([]byte, error) {
			return self.renderFile(entry.file)
		})
		if err != nil {
			return fmt.Errorf("failed to generate page for file %s: %w", entry.Path, err)
		}

		if entry.Raw != "" {
			if err := self.dispatchRawFile(entry); err != nil {
				return fmt.Errorf("failed to generate raw file %s: %w", entry.Path, err)
			}
		}
	}

	self.indexFile.SetProp("Repo", self.data)
	self.indexFile.SetProp("RepoRoot", "")
	self.context.DispatchFile(self.indexFile)
	return nil
}

func (self *generator) collectLog() error {
	iter, err := self.repo.Log(&git.LogOptions{From: self.head.Hash, Order: git.LogOrderCommitterTime})
	if err != nil {
		return err
	}
	defer iter.Close()

	return iter.ForEach(func(commit *object.Commit) error {
		if self.plugin.maxCommits > 0 && len(self.data.Commits) >= self.plugin.maxCommits {
			return storer.ErrStop
		}

		data := self.commit(commit)
		data.Page = "commit/" + data.Hash + ".html"
		self.data.Commits = append(self.data.Commits, data)
		return nil
	})
}

func (self *generator) collectRefs() error {
	refs, err := self.repo.References()
	if err != nil {
		return err
	}

	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Type() != plumbing.HashReference || !ref.Name().IsBranch() && !ref.Name().IsTag() {
			return nil
		}

		obj, err := self.repo.Object(plumbing.AnyObject, ref.Hash())
		if err != nil {
			return err
		}
		if tag, ok := obj.(*object.Tag); ok {
			if obj, err = tag.Object(); err != nil {
				return err
			}
		}
		commit, ok := obj.(*object.Commit)
		if !ok {
			return nil // refs to trees and blobs are not listed
		}

		data := &Ref{Name: ref.Name().Short(), Commit: self.commit(commit)}
		if ref.Name().IsBranch() {
			self.data.Branches = append(self.data.Branches, data)
		} else {
			self.data.Tags = append(self.data.Tags, data)
		}

		return nil
	})
	if err != nil {
		return err
	}

	sort.Slice(self.data.Branches, func(i, j int) bool {
		return self.data.Branches[i].Name < self.data.Branches[j].Name
	})
	sort.Slice(self.data.Tags, func(i, j int) bool {
		ti, tj := self.data.Tags[i], self.data.Tags[j]
		if !ti.Commit.AuthorDate.Equal(tj.Commit.AuthorDate) {
			return ti.Commit.AuthorDate.After(tj.Commit.AuthorDate)
		}
		return ti.Name < tj.Name
	})

	return nil
}

func (self *generator) collectFiles() error {
	tree, err := self.head.Tree()
	if err != nil {
		return err
	}

	return tree.Files().ForEach(func(file *object.File) error {
		entry := &Entry{
			Path: file.Name,
			Name: path.Base(file.Name),
			Mode: modeString(file.Mode),
			Size: file.Size,
			Hash: file.Hash.String(),
			Page: "file/" + escapePath(file.Name) + ".html",
			file: file,
		}
		self.data.Files = append(self.data.Files, entry)
		self.files[entry.Path] = entry

		if !strings.Contains(file.Name, "/") {
			stem := strings.ToUpper(strings.TrimSuffix(entry.Name, path.Ext(entry.Name)))
			switch {
			case stem == "README" && (self.data.Readme == nil || isMarkdown(entry.Name)):
				self.data.Readme = entry
			case (stem == "LICENSE" || stem == "LICENCE" || stem == "COPYING") && self.data.License == nil:
				self.data.License = entry
			}
		}

		return nil
	})
}

func isMarkdown(name string) bool {
	ext := strings.ToLower(path.Ext(name))
	return ext == ".md" || ext == ".markdown"
}

func (self *generator) renderReadme() error {
	readme := self.data.Readme
	if readme == nil || !isMarkdown(readme.Name) {
		return nil
	}

	source, err := readme.file.Contents()
	if err != nil {
		return err
	}

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Typographer),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithASTTransformers(util.Prioritized(readmeLinks{self}, 0)),
		),
	)

	var buff bytes.Buffer
	if err := md.Convert([]byte(source), &buff); err != nil {
		return err
	}

	self.data.ReadmeHTML = template.HTML(buff.String())
	return nil
}

// readmeLinks rewrites README links and images that refer to files in the
// tracked branch, marking image targets to be written as raw files.
type readmeLinks struct {
	gen *generator
}

func (self readmeLinks) Transform(doc *ast.Document, _ text.Reader, _ parser.Context) {
	ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := node.(type) {
		case *ast.Link:
			if entry := self.gen.resolveLink(node.Destination); entry != nil {
				node.Destination = []byte(entry.Page)
			}
		case *ast.Image:
			if entry := self.gen.resolveLink(node.Destination); entry != nil {
				entry.Raw = "raw/" + escapePath(entry.Path)
				node.Destination = []byte(entry.Raw)
			}
		}

		return ast.WalkContinue, nil
	})
}

// resolveLink finds the file referenced by a relative link from the top-level
// README, where paths beginning with "/" are relative to the repository root.
func (self *generator) resolveLink(destination []byte) *Entry {
	target, err := url.Parse(string(destination))
	if err != nil || target.Scheme != "" || target.Host != "" || target.Path == "" {
		return nil
	}

	return self.files[strings.TrimPrefix(path.Clean("/"+target.Path), "/")]
}

func (self *generator) dispatchRawFile(entry *Entry) error {
	reader, err := entry.file.Reader()
	if err != nil {
		return err
	}
	defer reader.Close()

	file, err := self.context.CreateFileFromReader(path.Join(self.data.Path, "raw", entry.Path), reader)
	if err != nil {
		return err
	}

	self.context.DispatchFile(file)
	return nil
}

// escapePath escapes each segment of a slash-separated path for use in URLs.
func escapePath(relPath string) string {
	segments := strings.Split(relPath, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}

	return strings.Join(segments, "/")
}

func modeString(mode filemode.FileMode) string {
	switch mode {
	case filemode.Executable:
		return "-rwxr-xr-x"
	case filemode.Symlink:
		return "lrwxrwxrwx"
	default:
		return "-rw-r--r--"
	}
}

// dispatchPage dispatches a generated page with content produced by render,
// which is nil for pages without content. If hash is not empty, the content
// depends only on that git object, and is cached by it.
func (self *generator) dispatchPage(relPath, layout, hash string, props goldsmith.FileProps, render func() ([]byte, error)) error {
	outputPath := path.Join(self.data.Path, relPath)

	var file *goldsmith.File
	if hash != "" {
		file = self.context.RetrieveCachedFile(outputPath, hash)
	}

	cached := file != nil
	if !cached {
		var body []byte
		if render != nil {
			var err error
			if body, err = render(); err != nil {
				return err
			}
		}

		var err error
		if file, err = self.context.CreateFileFromReader(outputPath, bytes.NewReader(body)); err != nil {
			return err
		}
	}

	file.SetProp("Repo", self.data)
	file.SetProp("RepoRoot", strings.Repeat("../", strings.Count(relPath, "/")))
	file.SetProp(self.plugin.layoutKey, layout)
	for name, value := range props {
		file.SetProp(name, value)
	}

	if cached || hash == "" {
		self.context.DispatchFile(file)
	} else {
		self.context.DispatchAndCacheFile(file, hash)
	}

	return nil
}

func (self *generator) renderFile(file *object.File) ([]byte, error) {
	if self.plugin.maxFileSize > 0 && file.Size > self.plugin.maxFileSize {
		return []byte("<p class=\"file-too-large\">File too large to display.</p>\n"), nil
	}

	binary, err := file.IsBinary()
	if err != nil {
		return nil, err
	}
	if binary {
		return []byte("<p class=\"file-binary\">Binary file.</p>\n"), nil
	}

	contents, err := file.Contents()
	if err != nil {
		return nil, err
	}

	var buff bytes.Buffer
	buff.WriteString("<pre><code")
	if lexer := lexers.Match(file.Name); lexer != nil {
		config := lexer.Config()
		language := strings.ToLower(config.Name)
		if len(config.Aliases) > 0 {
			language = config.Aliases[0]
		}
		fmt.Fprintf(&buff, ` class="language-%s"`, html.EscapeString(language))
	}
	buff.WriteString(">")
	buff.WriteString(html.EscapeString(contents))
	buff.WriteString("</code></pre>\n")

	return buff.Bytes(), nil
}

// renderDiff produces an HTML diffstat and unified diff for a commit against
// its first parent (or the empty tree for root commits).
func renderDiff(commit *object.Commit) ([]byte, error) {
	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	var parentTree *object.Tree
	if commit.NumParents() > 0 {
		parent, err := commit.Parent(0)
		if err != nil {
			return nil, err
		}
		if parentTree, err = parent.Tree(); err != nil {
			return nil, err
		}
	}

	patch, err := parentTree.Patch(tree)
	if err != nil {
		return nil, err
	}

	var encoded bytes.Buffer
	if err := patch.Encode(&encoded); err != nil {
		return nil, err
	}

	// The encoded patch has a "diff --git" header for each file patch, in order.
	type fileStat struct {
		name                 string
		additions, deletions int
	}
	var stats []fileStat
	for _, filePatch := range patch.FilePatches() {
		var name string
		switch from, to := filePatch.Files(); {
		case from == nil:
			name = to.Path()
		case to == nil || from.Path() == to.Path():
			name = from.Path()
		default:
			name = from.Path() + " => " + to.Path()
		}
		stats = append(stats, fileStat{name: name})
	}

	var diff bytes.Buffer
	file, inHeader := -1, false
	for _, line := range strings.SplitAfter(encoded.String(), "\n") {
		if line == "" {
			continue
		}

		content := strings.TrimSuffix(line, "\n")
		newline := line[len(content):]

		var class string
		switch {
		case strings.HasPrefix(line, "diff --git "):
			file++
			inHeader = true
			fmt.Fprintf(&diff, "<b id=\"f%d\">%s</b>%s", file, html.EscapeString(content), newline)
			continue
		case strings.HasPrefix(line, "@@"):
			class = "hunk"
			inHeader = false
		case inHeader:
			class = "head"
		case strings.HasPrefix(line, "+"):
			class = "add"
			stats[file].additions++
		case strings.HasPrefix(line, "-"):
			class = "del"
			stats[file].deletions++
		}

		if class == "" {
			diff.WriteString(html.EscapeString(line))
		} else {
			fmt.Fprintf(&diff, "<span class=\"%s\">%s</span>%s", class, html.EscapeString(content), newline)
		}
	}

	var buff bytes.Buffer
	var additions, deletions int
	buff.WriteString("<table class=\"diffstat\">\n")
	for i, stat := range stats {
		fmt.Fprintf(
			&buff,
			"<tr><td class=\"path\"><a href=\"#f%d\">%s</a></td><td class=\"add\">+%d</td><td class=\"del\">-%d</td></tr>\n",
			i, html.EscapeString(stat.name), stat.additions, stat.deletions,
		)
		additions += stat.additions
		deletions += stat.deletions
	}
	buff.WriteString("</table>\n")
	fmt.Fprintf(
		&buff,
		"<p class=\"diffsummary\">%d %s changed, %d %s(+), %d %s(-)</p>\n",
		len(stats), plural(len(stats), "file", "files"),
		additions, plural(additions, "insertion", "insertions"),
		deletions, plural(deletions, "deletion", "deletions"),
	)
	fmt.Fprintf(&buff, "<pre class=\"diff\">%s</pre>\n", diff.Bytes())

	return buff.Bytes(), nil
}

func plural(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}

	return plural
}
