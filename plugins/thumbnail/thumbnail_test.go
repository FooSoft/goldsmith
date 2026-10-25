package thumbnail

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.foosoft.net/goldsmith"
)

func Test(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	plugin := New()
	for range 2 {
		gs := goldsmith.Goldsmith{CacheDir: filepath.Join(root, "cache")}
		if errs := gs.Begin("testdata/source").Chain(plugin).End(target); len(errs) > 0 {
			t.Fatal(errs)
		}
		for _, name := range []string{"image_1.png", "image_2.jpg", "image_1-thumb.png", "image_2-thumb.png"} {
			actual, err := os.ReadFile(filepath.Join(target, name))
			if err != nil {
				t.Fatal(err)
			}
			expected, err := os.ReadFile(filepath.Join("testdata/reference", name))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(name, "-thumb") {
				if !bytes.Equal(actual, expected) {
					t.Errorf("source image %s changed", name)
				}
				continue
			}
			a, err := png.Decode(bytes.NewReader(actual))
			if err != nil {
				t.Fatal(err)
			}
			b, err := png.Decode(bytes.NewReader(expected))
			if err != nil {
				t.Fatal(err)
			}
			if a.Bounds() != b.Bounds() {
				t.Fatalf("%s bounds: got %v, want %v", name, a.Bounds(), b.Bounds())
			}
			// Compare pixels instead of PNG compression bytes. JPEG decoding
			// can vary by one color level between Go toolchains/architectures.
			tolerance := 0
			if name == "image_2-thumb.png" {
				tolerance = 1
			}
			for y := 0; y < a.Bounds().Dy(); y++ {
				for x := 0; x < a.Bounds().Dx(); x++ {
					got := color.NRGBAModel.Convert(a.At(x, y)).(color.NRGBA)
					want := color.NRGBAModel.Convert(b.At(x, y)).(color.NRGBA)
					for i, delta := range []int{int(got.R) - int(want.R), int(got.G) - int(want.G), int(got.B) - int(want.B), int(got.A) - int(want.A)} {
						if delta < -tolerance || delta > tolerance {
							t.Fatalf("%s pixel (%d,%d) channel %d: got %v, want %v", name, x, y, i, got, want)
						}
					}
				}
			}
		}
	}
}

func TestThumbnailStylesAndFormats(t *testing.T) {
	context := new(goldsmith.Context)
	img := image.NewNRGBA(image.Rect(0, 0, 40, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 40; x++ {
			img.Set(x, y, color.NRGBA{R: 255, A: 255})
		}
	}
	var input bytes.Buffer
	if err := png.Encode(&input, img); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		style Style
		ext   string
		want  image.Rectangle
	}{
		{"fit", Fit, ".png", image.Rect(0, 0, 10, 5)},
		{"crop", Crop, ".jpg", image.Rect(0, 0, 10, 10)},
		{"pad", Pad, ".gif", image.Rect(0, 0, 10, 10)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file, err := context.CreateFileFromReader("input.png", bytes.NewReader(input.Bytes()))
			if err != nil {
				t.Fatal(err)
			}
			file.SetProp("title", "picture")
			output, err := New().Size(10).Style(tc.style).Color(color.White).thumbnail(context, file, "thumb"+tc.ext)
			if err != nil {
				t.Fatal(err)
			}
			if output.PropOrDef("title", "") != "picture" {
				t.Fatal("thumbnail lost metadata")
			}
			data, err := io.ReadAll(output)
			if err != nil {
				t.Fatal(err)
			}
			result, _, err := image.Decode(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			if result.Bounds() != tc.want {
				t.Fatalf("got %v, want %v", result.Bounds(), tc.want)
			}
			if tc.style == Pad && result.At(0, 0) != color.White {
				r, g, b, a := result.At(0, 0).RGBA()
				if r != 65535 || g != 65535 || b != 65535 || a != 65535 {
					t.Fatal("padding color was not applied")
				}
			}
		})
	}
}

func TestRejectsInvalidConfiguration(t *testing.T) {
	if err := New().Size(0).Initialize(new(goldsmith.Context)); err == nil {
		t.Fatal("zero thumbnail size was accepted")
	}
	if err := New().Namer(nil).Initialize(new(goldsmith.Context)); err == nil {
		t.Fatal("nil thumbnail namer was accepted")
	}
	if err := New().Style(Style(99)).Initialize(new(goldsmith.Context)); err == nil {
		t.Fatal("invalid thumbnail style was accepted")
	}
	if err := New().Style(Pad).Color(nil).Initialize(new(goldsmith.Context)); err == nil {
		t.Fatal("nil thumbnail padding color was accepted")
	}
}

func TestCachedThumbnailTracksSize(t *testing.T) {
	root := t.TempDir()
	for _, size := range []int{16, 32} {
		gs := goldsmith.Goldsmith{CacheDir: filepath.Join(root, "cache")}
		target := filepath.Join(root, "target")
		if errs := gs.Begin("testdata/source").Chain(New().Size(size)).End(target); len(errs) > 0 {
			t.Fatal(errs)
		}
		file, err := os.Open(filepath.Join(target, "image_1-thumb.png"))
		if err != nil {
			t.Fatal(err)
		}
		config, err := png.DecodeConfig(file)
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
		if max(config.Width, config.Height) != size {
			t.Fatalf("cached thumbnail: got %dx%d, want maximum dimension %d", config.Width, config.Height, size)
		}
	}
}
