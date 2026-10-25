package goldsmith

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileReaderAndMetadata(t *testing.T) {
	context := new(Context)
	file, err := context.CreateFileFromReader("dir/page.txt", strings.NewReader("abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	if file.Dir() != "dir" || file.Name() != "page.txt" || file.Ext() != ".txt" || file.Size() != 6 {
		t.Fatalf("incorrect file attributes: %#v", file)
	}
	file.SetProp("title", "page")
	copy, err := context.CreateFileFromReader("copy.txt", strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	copy.CopyProps(file)
	copy.RemoveProp("title")
	if file.PropOrDef("title", "missing") != "page" || copy.PropOrDef("title", "missing") != "missing" {
		t.Fatal("metadata copy was not independent")
	}
	if _, err := file.Seek(-3, io.SeekEnd); err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "def" {
		t.Fatalf("got %q", data)
	}
	if _, err := file.Seek(-1, io.SeekStart); err == nil {
		t.Fatal("negative seek was accepted")
	}
}

func TestFileAssetValidationAndLazyLoading(t *testing.T) {
	context := new(Context)
	root := t.TempDir()
	if _, err := context.CreateFileFromAsset("file", root); err == nil {
		t.Fatal("directory accepted as file")
	}
	asset := filepath.Join(root, "asset")
	if err := os.WriteFile(asset, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	file, err := context.CreateFileFromAsset("file", asset)
	if err != nil {
		t.Fatal(err)
	}
	if file.reader != nil {
		t.Fatal("asset was eagerly loaded")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	if file.reader != nil {
		t.Fatal("rewinding unloaded asset read it")
	}
	if err := os.Remove(asset); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadAll(file); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("got %v, want missing asset error", err)
	}
}
