package goldsmith

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestContextCacheIsolation(t *testing.T) {
	chain := &chainState{cache: &cache{baseDir: t.TempDir()}}
	newContext := func(name string, index int, key string) *Context {
		return (&Context{chain: chain, plugin: cachePluginName(name), index: index}).CacheKey(key)
	}
	context := newContext("first", 1, "config")
	input, err := context.CreateFileFromReader("input.txt", strings.NewReader("input"))
	if err != nil {
		t.Fatal(err)
	}
	output, err := context.CreateFileFromReader("output.txt", strings.NewReader("cached output"))
	if err != nil {
		t.Fatal(err)
	}
	output.SetProp("title", "not cached")
	context.filesOut = make(chan *File, 1)
	context.DispatchAndCacheFile(output, FileCacheKeys(input)...)
	if got := <-context.filesOut; got != output {
		t.Fatal("did not dispatch output")
	}
	for _, tc := range []struct {
		name    string
		context *Context
		wantHit bool
	}{
		{"same configuration", newContext("first", 1, "config"), true},
		{"different name", newContext("second", 1, "config"), false},
		{"different position", newContext("first", 2, "config"), false},
		{"different configuration", newContext("first", 1, "other"), false},
		{"different plugin type", (&Context{chain: chain, plugin: otherCachePlugin{}, index: 1}).CacheKey("config"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cached := tc.context.RetrieveCachedFile(output.Path(), FileCacheKeys(input)...)
			if (cached != nil) != tc.wantHit {
				t.Fatalf("cache hit = %t, want %t", cached != nil, tc.wantHit)
			}
			if cached == nil {
				return
			}
			data, err := io.ReadAll(cached)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != "cached output" {
				t.Fatalf("got %q", data)
			}
			if len(cached.Props()) != 0 {
				t.Fatal("cache unexpectedly included metadata")
			}
		})
	}
	input.modTime = input.modTime.Add(time.Second)
	if context.RetrieveCachedFile(output.Path(), FileCacheKeys(input)...) != nil {
		t.Fatal("modified dependency hit cache")
	}
	if len(chain.errors) != 0 {
		t.Fatal(chain.errors)
	}
}

type cachePluginName string

func TestContextCacheKeys(t *testing.T) {
	chain := &chainState{cache: &cache{baseDir: t.TempDir()}}
	context := &Context{chain: chain, plugin: cachePluginName("git"), filesOut: make(chan *File, 2)}
	output, err := context.CreateFileFromReader("commit/abc.html", strings.NewReader("diff"))
	if err != nil {
		t.Fatal(err)
	}
	context.DispatchAndCacheFile(output, "abc", "def")
	if got := <-context.filesOut; got != output {
		t.Fatal("did not dispatch output")
	}

	cached := context.RetrieveCachedFile("commit/abc.html", "abc", "def")
	if cached == nil {
		t.Fatal("expected cache hit")
	}
	if data, err := io.ReadAll(cached); err != nil || string(data) != "diff" {
		t.Fatalf("got %q, %v", data, err)
	}

	for name, keys := range map[string][]string{
		"different key":  {"abc", "xyz"},
		"reordered keys": {"def", "abc"},
		"extra key":      {"abc", "def", ""},
		"no keys":        nil,
	} {
		if context.RetrieveCachedFile("commit/abc.html", keys...) != nil {
			t.Errorf("%s: unexpected cache hit", name)
		}
	}

	if len(chain.errors) != 0 {
		t.Fatal(chain.errors)
	}
}

func (p cachePluginName) Name() string { return string(p) }

type otherCachePlugin struct{}

func (otherCachePlugin) Name() string { return "first" }

func TestCacheRetrievalErrors(t *testing.T) {
	root := t.TempDir()
	broken := filepath.Join(root, "file")
	if err := os.WriteFile(broken, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	context := &Context{chain: &chainState{cache: &cache{baseDir: broken}}, plugin: cachePluginName("test")}
	if context.RetrieveCachedFile("output.txt") != nil {
		t.Fatal("unexpected cached file")
	}
	if len(context.chain.errors) != 1 {
		t.Fatalf("expected one error, got %v", context.chain.errors)
	}
	var pathErr *os.PathError
	if !errors.As(context.chain.errors[0], &pathErr) {
		t.Fatalf("file error was not wrapped: %v", context.chain.errors)
	}
}
