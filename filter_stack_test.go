package goldsmith

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestFilterScopesAndPassthrough(t *testing.T) {
	root := t.TempDir()
	source, target := filepath.Join(root, "source"), filepath.Join(root, "target")
	if err := os.Mkdir(source, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(source, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	first := new(filterRecorder)
	second := &filterRecorder{filter: onlyPath("b.txt")}
	third := new(filterRecorder)
	var gs Goldsmith
	errs := gs.Begin(source).
		FilterPush(onlyPath("a.txt")).Chain(first).FilterPop().
		FilterPush(onlyPath("b.txt")).Chain(second).FilterPop().
		Chain(third).End(target)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	for _, tc := range []struct {
		plugin *filterRecorder
		want   []string
	}{
		{first, []string{"a.txt"}},
		{second, []string{"b.txt"}},
		{third, []string{"a.txt", "b.txt"}},
	} {
		slices.Sort(tc.plugin.files)
		if !slices.Equal(tc.plugin.files, tc.want) {
			t.Errorf("processed %v, want %v", tc.plugin.files, tc.want)
		}
	}
	for _, name := range []string{"a.txt", "b.txt"} {
		data, err := os.ReadFile(filepath.Join(target, name))
		if err != nil || string(data) != name {
			t.Errorf("passthrough output %s: %q, %v", name, data, err)
		}
	}
}

type onlyPath string

func (onlyPath) Name() string { return "only-path" }

func (p onlyPath) Accept(file *File) bool { return file.Path() == string(p) }

type filterRecorder struct {
	filter Filter
	files  []string
}

func (*filterRecorder) Name() string { return "filter-recorder" }

func (p *filterRecorder) Initialize(context *Context) error {
	context.Threads(1)
	if p.filter != nil {
		context.Filter(p.filter)
	}
	return nil
}

func (p *filterRecorder) Process(context *Context, file *File) error {
	p.files = append(p.files, file.Path())
	context.DispatchFile(file)
	return nil
}
