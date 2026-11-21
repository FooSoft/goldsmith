package gitrepo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/harness"
	"go.foosoft.net/goldsmith/plugins/frontmatter"
	"go.foosoft.net/goldsmith/plugins/layout"
	"go.foosoft.net/goldsmith/plugins/markdown"
)

type testRepo struct {
	t        *testing.T
	dir      string
	repo     *git.Repository
	worktree *git.Worktree
	when     time.Time
}

func newTestRepo(t *testing.T, dir string) *testRepo {
	t.Helper()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	return &testRepo{t: t, dir: dir, repo: repo, worktree: worktree, when: time.Date(2024, 1, 2, 3, 4, 0, 0, time.UTC)}
}

func (self *testRepo) signature() *object.Signature {
	return &object.Signature{Name: "Jane Doe", Email: "jane@example.com", When: self.when}
}

func (self *testRepo) commit(message string, files map[string]string, removed ...string) plumbing.Hash {
	self.t.Helper()
	for name, data := range files {
		path := filepath.Join(self.dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			self.t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			self.t.Fatal(err)
		}
	}
	for _, name := range removed {
		if _, err := self.worktree.Remove(name); err != nil {
			self.t.Fatal(err)
		}
	}
	if err := self.worktree.AddWithOptions(&git.AddOptions{All: true}); err != nil {
		self.t.Fatal(err)
	}

	hash, err := self.worktree.Commit(message, &git.CommitOptions{Author: self.signature(), Committer: self.signature()})
	if err != nil {
		self.t.Fatal(err)
	}

	self.when = self.when.Add(time.Hour)
	return hash
}

func (self *testRepo) checkout(branch string, create bool) {
	self.t.Helper()
	err := self.worktree.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(branch), Create: create})
	if err != nil {
		self.t.Fatal(err)
	}
}

const readme = `# Demo

![logo](./logo.bin) ![missing](missing.png) ![remote](https://example.com/logo.png)

- [guide](docs/guide/run.txt)
- [main](/main.go#L1)
- [escaped](../../LICENSE)
- [percent](50%25%20%231.txt)
- [deleted](docs/a%20b.txt)
- [web](https://example.com/)
- [mail](mailto:jane@example.com)
- [top](#demo)
- [script](javascript:alert(1))

Inline <b>html</b>.

<div onclick="alert(1)">block html</div>
`

func createDemoRepo(t *testing.T, dir string) *testRepo {
	repo := newTestRepo(t, dir)
	first := repo.commit("Initial commit\n\nAdd the basics.\n", map[string]string{
		"README":             "Plain text README\n",
		"README.md":          readme,
		"LICENSE":            "MIT\n",
		"main.go":            "package main\n\nfunc main() {}\n",
		"docs/a b.txt":       "<hello & goodbye>\n",
		"docs/guide/run.txt": "go run .\n",
		"logo.bin":           "\x00\x01\x02",
		"50% #1.txt":         "fifty\n",
		"notes.txt":          "one\ntwo\nthree\n",
	})
	if _, err := repo.repo.CreateTag("v0.1", first, &git.CreateTagOptions{Tagger: repo.signature(), Message: "v0.1"}); err != nil {
		t.Fatal(err)
	}

	second := repo.commit("Print a greeting\n", map[string]string{
		"main.go":        "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hi\")\n}\n",
		"logo.bin":       "\x00\x03",
		"docs/notes.txt": "one\ntwo\nthree\n",
	}, "docs/a b.txt", "notes.txt")
	if _, err := repo.repo.CreateTag("v0.2", second, nil); err != nil {
		t.Fatal(err)
	}

	repo.checkout("feature", true)
	repo.commit("Add feature\n", map[string]string{"feature.txt": "feature\n"})
	repo.checkout("master", false)

	for name, data := range map[string]string{"description": "Demo repository\n", "url": "https://example.com/demo.git\n"} {
		if err := os.WriteFile(filepath.Join(dir, ".git", name), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return repo
}

func Test(t *testing.T) {
	rootDir := t.TempDir()
	createDemoRepo(t, filepath.Join(rootDir, "demo"))

	harness.Validate(t, func(gs *goldsmith.Goldsmith) {
		gs.
			Chain(frontmatter.New()).
			Chain(New(rootDir)).
			Chain(markdown.New()).
			Chain(layout.New())
	})
}

func build(t *testing.T, sourceDir, targetDir, cacheDir string, plugin *GitRepo) {
	t.Helper()
	gs := goldsmith.Goldsmith{CacheDir: cacheDir, Clean: true}
	gs.Begin(sourceDir).Chain(frontmatter.New()).Chain(plugin)
	if errs := gs.End(targetDir); len(errs) > 0 {
		t.Fatal(errs)
	}
}

func TestBranchLimitsAndUpdates(t *testing.T) {
	var (
		rootDir   = t.TempDir()
		sourceDir = filepath.Join(rootDir, "source")
		targetDir = filepath.Join(rootDir, "target")
		cacheDir  = filepath.Join(rootDir, "cache")
	)
	repo := createDemoRepo(t, filepath.Join(rootDir, "demo"))
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "index.html"), []byte("---\nRepository: demo\nBranch: feature\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	countPages := func(dir string) int {
		entries, _ := os.ReadDir(filepath.Join(targetDir, dir))
		return len(entries)
	}

	plugin := New(rootDir).MaxCommits(2)
	build(t, sourceDir, targetDir, cacheDir, plugin)
	if _, err := os.Stat(filepath.Join(targetDir, "file", "feature.txt.html")); err != nil {
		t.Fatal("branch override was not used:", err)
	}
	if count := countPages("commit"); count != 2 {
		t.Fatalf("got %d commit pages, want 2", count)
	}

	repo.checkout("feature", false)
	newHash := repo.commit("Update feature\n", map[string]string{"feature.txt": "feature <v2>\n"})
	build(t, sourceDir, targetDir, cacheDir, plugin)

	data, err := os.ReadFile(filepath.Join(targetDir, "commit", newHash.String()+".html"))
	if err != nil {
		t.Fatal("new commit page missing:", err)
	}
	if !strings.Contains(string(data), `<span class="add">+feature &lt;v2&gt;</span>`) {
		t.Fatalf("unexpected diff:\n%s", data)
	}
	data, err = os.ReadFile(filepath.Join(targetDir, "file", "feature.txt.html"))
	if err != nil || !strings.Contains(string(data), "feature &lt;v2&gt;") {
		t.Fatalf("stale file page (%v):\n%s", err, data)
	}
	if count := countPages("commit"); count != 2 {
		t.Fatalf("got %d commit pages, want 2", count)
	}
}

func TestInvalidRepository(t *testing.T) {
	rootDir := t.TempDir()
	sourceDir := filepath.Join(rootDir, "source")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "index.html"), []byte("---\nRepository: missing\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	gs := goldsmith.Goldsmith{Clean: true}
	gs.Begin(sourceDir).Chain(frontmatter.New()).Chain(New(rootDir))
	if errs := gs.End(filepath.Join(rootDir, "target")); len(errs) == 0 {
		t.Fatal("expected error for missing repository")
	}
}

func TestEmptyRepository(t *testing.T) {
	var (
		rootDir   = t.TempDir()
		sourceDir = filepath.Join(rootDir, "source")
		targetDir = filepath.Join(rootDir, "target")
	)
	newTestRepo(t, filepath.Join(rootDir, "empty"))
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "index.html"), []byte("---\nRepository: empty\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	build(t, sourceDir, targetDir, "", New(rootDir))
	for _, name := range []string{"index.html", "log.html", "files.html", "refs.html"} {
		if _, err := os.Stat(filepath.Join(targetDir, name)); err != nil {
			t.Error(err)
		}
	}
}
