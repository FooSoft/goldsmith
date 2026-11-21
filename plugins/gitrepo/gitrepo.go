// Package gitrepo generates static pages for browsing git repositories, in the
// style of stagit. Files with a "Repository" metadata property (typically set
// via front matter) are annotated with repository metadata, and log, file
// listing, refs, commit, and file pages are generated alongside them. Generated
// pages are HTML files with a "Layout" property, intended to be rendered by the
// "layout" plugin. Commit diffs and file contents are provided as page content
// (available to templates as "Content") and are cached by git object hash, so
// only new commits and changed files are processed on subsequent builds.
//
// A single branch is tracked per repository: the branch referenced by HEAD,
// unless overridden with Branch or a "Branch" metadata property. For example,
// given "projects/demo/index.md" containing:
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
// Every page, including the source page, has a "Repo" property (*Repository)
// and a "RepoRoot" property containing the relative path back to the
// repository directory (e.g. "../" on commit pages), for building links such as
// {{.Props.RepoRoot}}{{.Page}}. Commit pages also have a "Commit" property
// (*Commit) and file pages an "Entry" property (*Entry).
//
// When using devserver, include the repository directories in the watch list
// so that new commits trigger a rebuild.
package gitrepo

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/alecthomas/chroma/lexers"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	fdiff "github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"

	"go.foosoft.net/goldsmith"
)

// Repository contains metadata about a repository and its tracked branch.
type Repository struct {
	Name        string
	Description string
	CloneURL    string
	Branch      string
	Path        string // site-relative directory containing the repository pages

	Head     *Commit
	Commits  []*Commit // newest first, limited by MaxCommits
	Branches []*Ref
	Tags     []*Ref // newest first
	Files    []*Entry
	Readme   *Entry
	License  *Entry
}

// Commit contains metadata about a single commit.
type Commit struct {
	Hash      string
	ShortHash string
	Parents   []string

	Author         string
	AuthorEmail    string
	AuthorDate     time.Time
	Committer      string
	CommitterEmail string
	CommitterDate  time.Time

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

	file *object.File
}

// GitRepo chainable context.
type GitRepo struct {
	rootDir     string
	repoKey     string
	branchKey   string
	branch      string
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

// Branch sets the default tracked branch (default: the branch referenced by HEAD).
func (self *GitRepo) Branch(branch string) *GitRepo {
	self.branch = branch
	return self
}

// LayoutKey sets the metadata key used to assign layouts to generated pages (default: "Layout").
func (self *GitRepo) LayoutKey(key string) *GitRepo {
	self.layoutKey = key
	return self
}

// MaxCommits limits the number of commits listed in the log and given commit
// pages; zero means no limit (default: 0).
func (self *GitRepo) MaxCommits(count int) *GitRepo {
	self.maxCommits = count
	return self
}

// MaxFileSize sets the largest file size, in bytes, whose contents are
// rendered on file pages; zero means no limit (default: 1 MiB).
func (self *GitRepo) MaxFileSize(size int64) *GitRepo {
	self.maxFileSize = size
	return self
}

func (*GitRepo) Name() string {
	return "gitrepo"
}

func (self *GitRepo) Initialize(context *goldsmith.Context) error {
	if self.maxCommits < 0 || self.maxFileSize < 0 {
		return errors.New("gitrepo limits cannot be negative")
	}

	context.CacheKey(fmt.Sprintf("%d", self.maxFileSize))
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

	branch := self.branch
	if branchProp, ok := inputFile.Prop(self.branchKey); ok {
		if branch, ok = branchProp.(string); !ok || branch == "" {
			return fmt.Errorf("invalid %s property: %v", self.branchKey, branchProp)
		}
	}

	gen, err := newGenerator(self, context, inputFile, repoPath, branch)
	if err != nil {
		return err
	}

	return gen.generate()
}

type generator struct {
	plugin    *GitRepo
	context   *goldsmith.Context
	indexFile *goldsmith.File
	repo      *git.Repository
	head      *object.Commit
	data      *Repository
	commits   map[plumbing.Hash]*Commit
}

func newGenerator(plugin *GitRepo, context *goldsmith.Context, indexFile *goldsmith.File, repoPath, branch string) (*generator, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository %s: %w", repoPath, err)
	}

	var ref *plumbing.Reference
	if branch == "" {
		if ref, err = repo.Head(); err == nil && !ref.Name().IsBranch() {
			err = errors.New("HEAD is not a branch")
		}
	} else {
		ref, err = repo.Reference(plumbing.NewBranchReferenceName(branch), true)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to resolve branch in %s: %w", repoPath, err)
	}

	head, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, err
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
	cloneURL := readMetaFile(gitDir, "cloneurl")
	if cloneURL == "" {
		cloneURL = readMetaFile(gitDir, "url")
	}

	gen := &generator{
		plugin:    plugin,
		context:   context,
		indexFile: indexFile,
		repo:      repo,
		head:      head,
		commits:   make(map[plumbing.Hash]*Commit),
		data: &Repository{
			Name:        strings.TrimSuffix(name, ".git"),
			Description: description,
			CloneURL:    cloneURL,
			Branch:      ref.Name().Short(),
			Path:        indexFile.Dir(),
		},
	}
	gen.data.Head = gen.commit(head)

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
		Hash:           commit.Hash.String(),
		ShortHash:      commit.Hash.String()[:7],
		Author:         commit.Author.Name,
		AuthorEmail:    commit.Author.Email,
		AuthorDate:     commit.Author.When,
		Committer:      commit.Committer.Name,
		CommitterEmail: commit.Committer.Email,
		CommitterDate:  commit.Committer.When,
		Subject:        strings.TrimSpace(subject),
		Message:        commit.Message,
		object:         commit,
	}
	for _, parent := range commit.ParentHashes {
		data.Parents = append(data.Parents, parent.String())
	}

	self.commits[commit.Hash] = data
	return data
}

func (self *generator) generate() error {
	if err := self.collectLog(); err != nil {
		return err
	}
	if err := self.collectRefs(); err != nil {
		return err
	}
	if err := self.collectFiles(); err != nil {
		return err
	}

	for _, page := range []struct{ path, layout string }{
		{"log.html", "gitlog"},
		{"files.html", "gitfiles"},
		{"refs.html", "gitrefs"},
	} {
		file, err := self.createPage(page.path, nil)
		if err != nil {
			return err
		}

		self.setPageProps(file, page.path, page.layout, nil)
		self.context.DispatchFile(file)
	}

	for _, commit := range self.data.Commits {
		err := self.dispatchCachedPage(commit.Page, "gitcommit", commit.Hash, goldsmith.FileProps{"Commit": commit}, func() ([]byte, error) {
			return renderDiff(commit.object)
		})
		if err != nil {
			return fmt.Errorf("failed to generate page for commit %s: %w", commit.Hash, err)
		}
	}

	for _, entry := range self.data.Files {
		err := self.dispatchCachedPage("file/"+entry.Path+".html", "gitfile", entry.Hash, goldsmith.FileProps{"Entry": entry}, func() ([]byte, error) {
			return self.renderFile(entry.file)
		})
		if err != nil {
			return fmt.Errorf("failed to generate page for file %s: %w", entry.Path, err)
		}
	}

	self.setCommonProps(self.indexFile, "")
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
	branches, err := self.repo.Branches()
	if err != nil {
		return err
	}
	err = branches.ForEach(func(ref *plumbing.Reference) error {
		commit, err := self.repo.CommitObject(ref.Hash())
		if err != nil {
			return err
		}

		self.data.Branches = append(self.data.Branches, &Ref{Name: ref.Name().Short(), Commit: self.commit(commit)})
		return nil
	})
	if err != nil {
		return err
	}

	tags, err := self.repo.Tags()
	if err != nil {
		return err
	}
	err = tags.ForEach(func(ref *plumbing.Reference) error {
		var commit *object.Commit
		if tag, err := self.repo.TagObject(ref.Hash()); err == nil {
			if commit, err = tag.Commit(); err != nil {
				return nil // tags of non-commit objects are not listed
			}
		} else if commit, err = self.repo.CommitObject(ref.Hash()); err != nil {
			return nil
		}

		self.data.Tags = append(self.data.Tags, &Ref{Name: ref.Name().Short(), Commit: self.commit(commit)})
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

		if !strings.Contains(file.Name, "/") {
			stem := strings.ToUpper(strings.TrimSuffix(entry.Name, path.Ext(entry.Name)))
			switch {
			case stem == "README" && self.data.Readme == nil:
				self.data.Readme = entry
			case (stem == "LICENSE" || stem == "LICENCE" || stem == "COPYING") && self.data.License == nil:
				self.data.License = entry
			}
		}

		return nil
	})
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

func (self *generator) setCommonProps(file *goldsmith.File, relPath string) {
	file.SetProp("Repo", self.data)
	file.SetProp("RepoRoot", strings.Repeat("../", strings.Count(relPath, "/")))
}

func (self *generator) setPageProps(file *goldsmith.File, relPath, layout string, props goldsmith.FileProps) {
	self.setCommonProps(file, relPath)
	file.SetProp(self.plugin.layoutKey, layout)
	for name, value := range props {
		file.SetProp(name, value)
	}
}

func (self *generator) createPage(relPath string, body []byte) (*goldsmith.File, error) {
	return self.context.CreateFileFromReader(path.Join(self.data.Path, relPath), bytes.NewReader(body))
}

// dispatchCachedPage dispatches a page whose content depends only on the
// provided git object hash, rendering it only if it is not already cached.
func (self *generator) dispatchCachedPage(relPath, layout, hash string, props goldsmith.FileProps, render func() ([]byte, error)) error {
	if file := self.context.RetrieveCachedFileByKey(path.Join(self.data.Path, relPath), hash); file != nil {
		self.setPageProps(file, relPath, layout, props)
		self.context.DispatchFile(file)
		return nil
	}

	body, err := render()
	if err != nil {
		return err
	}

	file, err := self.createPage(relPath, body)
	if err != nil {
		return err
	}

	self.setPageProps(file, relPath, layout, props)
	self.context.DispatchAndCacheFileByKey(file, hash)
	return nil
}

func (self *generator) renderFile(file *object.File) ([]byte, error) {
	var buff bytes.Buffer
	if self.plugin.maxFileSize > 0 && file.Size > self.plugin.maxFileSize {
		buff.WriteString("<p class=\"file-too-large\">File too large to display.</p>\n")
		return buff.Bytes(), nil
	}

	binary, err := file.IsBinary()
	if err != nil {
		return nil, err
	}
	if binary {
		buff.WriteString("<p class=\"file-binary\">Binary file.</p>\n")
		return buff.Bytes(), nil
	}

	contents, err := file.Contents()
	if err != nil {
		return nil, err
	}

	buff.WriteString("<pre><code")
	if lexer := lexers.Match(path.Base(file.Name)); lexer != nil {
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

	var stats, diff, encoded bytes.Buffer
	var files, additions, deletions int
	for _, filePatch := range patch.FilePatches() {
		from, to := filePatch.Files()
		if from == nil && to == nil {
			continue
		}

		var name string
		switch {
		case from == nil:
			name = to.Path()
		case to == nil || from.Path() == to.Path():
			name = from.Path()
		default:
			name = from.Path() + " => " + to.Path()
		}

		var fileAdditions, fileDeletions int
		for _, chunk := range filePatch.Chunks() {
			switch chunk.Type() {
			case fdiff.Add:
				fileAdditions += countLines(chunk.Content())
			case fdiff.Delete:
				fileDeletions += countLines(chunk.Content())
			}
		}

		fmt.Fprintf(
			&stats,
			"<tr><td class=\"path\"><a href=\"#f%d\">%s</a></td><td class=\"add\">+%d</td><td class=\"del\">-%d</td></tr>\n",
			files, html.EscapeString(name), fileAdditions, fileDeletions,
		)

		encoded.Reset()
		if err := fdiff.NewUnifiedEncoder(&encoded, fdiff.DefaultContextLines).Encode(singlePatch{filePatch}); err != nil {
			return nil, err
		}

		inHeader := true
		for i, line := range strings.SplitAfter(encoded.String(), "\n") {
			if line == "" {
				continue
			}

			content := strings.TrimSuffix(line, "\n")
			newline := line[len(content):]

			var class string
			switch {
			case i == 0:
				fmt.Fprintf(&diff, "<b id=\"f%d\">%s</b>%s", files, html.EscapeString(content), newline)
				continue
			case strings.HasPrefix(line, "@@"):
				class = "hunk"
				inHeader = false
			case inHeader:
				class = "head"
			case strings.HasPrefix(line, "+"):
				class = "add"
			case strings.HasPrefix(line, "-"):
				class = "del"
			}

			if class == "" {
				diff.WriteString(html.EscapeString(line))
			} else {
				fmt.Fprintf(&diff, "<span class=\"%s\">%s</span>%s", class, html.EscapeString(content), newline)
			}
		}

		files++
		additions += fileAdditions
		deletions += fileDeletions
	}

	var buff bytes.Buffer
	fmt.Fprintf(&buff, "<table class=\"diffstat\">\n%s</table>\n", stats.Bytes())
	fmt.Fprintf(
		&buff,
		"<p class=\"diffsummary\">%d %s changed, %d %s(+), %d %s(-)</p>\n",
		files, plural(files, "file", "files"),
		additions, plural(additions, "insertion", "insertions"),
		deletions, plural(deletions, "deletion", "deletions"),
	)
	fmt.Fprintf(&buff, "<pre class=\"diff\">%s</pre>\n", diff.Bytes())

	return buff.Bytes(), nil
}

type singlePatch struct {
	filePatch fdiff.FilePatch
}

func (self singlePatch) FilePatches() []fdiff.FilePatch {
	return []fdiff.FilePatch{self.filePatch}
}

func (singlePatch) Message() string {
	return ""
}

func countLines(content string) int {
	if content == "" {
		return 0
	}

	count := strings.Count(content, "\n")
	if !strings.HasSuffix(content, "\n") {
		count++
	}

	return count
}

func plural(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}

	return plural
}
