// Package thumbnail automatically generates thumbnails for a variety of common
// image formats and saves them to a user-configurable path.
package thumbnail

import (
	"bytes"
	"errors"
	"fmt"
	"image/color"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/filters/wildcard"
)

// Namer callback function builds thumbnail file paths based on the original file path.
// An empty path can be returned if a thumbnail should not be generated for the current file.
type Namer func(string, int) string

// Style identifies a thumbnail resizing style.
type Style int

const (
	Fit Style = iota
	Crop
	Pad
)

// Thumbnail chainable context.
type Thumbnail struct {
	size  int
	style Style
	color color.Color
	namer Namer
}

// New creates a new instance of the Thumbnail plugin.
func New() *Thumbnail {
	namer := func(path string, dims int) string {
		ext := filepath.Ext(path)
		body := strings.TrimSuffix(path, ext)
		return fmt.Sprintf("%s-thumb.png", body)
	}

	return &Thumbnail{
		size:  128,
		style: Fit,
		color: color.Transparent,
		namer: namer,
	}
}

// Size sets the desired maximum pixel size of generated thumbnails (default: 128).
func (self *Thumbnail) Size(dims int) *Thumbnail {
	self.size = dims
	return self
}

// Style sets the desired thumbnailing style.
func (self *Thumbnail) Style(style Style) *Thumbnail {
	self.style = style
	return self
}

// Color sets the background color used by padded thumbnails (default: transparent).
func (self *Thumbnail) Color(color color.Color) *Thumbnail {
	self.color = color
	return self
}

// Namer sets the callback used to build paths for thumbnail files.
// Default naming appends "-thumb" to the path and changes extension to PNG,
// for example "file.jpg" becomes "file-thumb.png".
func (self *Thumbnail) Namer(namer Namer) *Thumbnail {
	self.namer = namer
	return self
}

func (*Thumbnail) Name() string {
	return "thumbnail"
}

func (self *Thumbnail) Initialize(context *goldsmith.Context) error {
	if self.size <= 0 {
		return errors.New("thumbnail size must be greater than zero")
	}
	if self.namer == nil {
		return errors.New("thumbnail namer cannot be nil")
	}
	if self.style != Fit && self.style != Crop && self.style != Pad {
		return errors.New("unsupported thumbnailing style")
	}
	if self.style == Pad && self.color == nil {
		return errors.New("thumbnail padding color cannot be nil")
	}

	context.Filter(wildcard.New("**/*.jpg", "**/*.jpeg", "**/*.gif", "**/*.png"))
	var r, g, b, a uint32
	if self.style == Pad {
		r, g, b, a = self.color.RGBA()
	}
	context.CacheKey(fmt.Sprintf("%d/%d/%d/%d/%d/%d", self.size, self.style, r, g, b, a))
	return nil
}

func (self *Thumbnail) Process(context *goldsmith.Context, inputFile *goldsmith.File) error {
	defer context.DispatchFile(inputFile)

	thumbPath := self.namer(inputFile.Path(), self.size)
	if len(thumbPath) == 0 {
		return nil
	}

	if outputFile := context.RetrieveCachedFile(thumbPath, goldsmith.FileCacheKeys(inputFile)...); outputFile != nil {
		outputFile.CopyProps(inputFile)
		context.DispatchFile(outputFile)
		return nil
	}

	outputFile, err := self.thumbnail(context, inputFile, thumbPath)
	if err != nil {
		return err
	}

	context.DispatchAndCacheFile(outputFile, goldsmith.FileCacheKeys(inputFile)...)
	return nil
}

func (self *Thumbnail) thumbnail(context *goldsmith.Context, inputFile *goldsmith.File, thumbPath string) (*goldsmith.File, error) {
	var thumbFormat imaging.Format
	switch strings.ToLower(filepath.Ext(thumbPath)) {
	case ".jpg", ".jpeg":
		thumbFormat = imaging.JPEG
	case ".gif":
		thumbFormat = imaging.GIF
	case ".png":
		thumbFormat = imaging.PNG
	default:
		return nil, errors.New("unsupported image format")
	}

	thumbImage, err := imaging.Decode(inputFile)
	if err != nil {
		return nil, err
	}

	switch self.style {
	case Fit:
		thumbImage = imaging.Fit(thumbImage, self.size, self.size, imaging.Lanczos)
	case Crop:
		thumbImage = imaging.Fill(thumbImage, self.size, self.size, imaging.Center, imaging.Lanczos)
	case Pad:
		thumbImage = imaging.Fit(thumbImage, self.size, self.size, imaging.Lanczos)
		thumbImage = imaging.PasteCenter(imaging.New(self.size, self.size, self.color), thumbImage)
	default:
		return nil, errors.New("unsupported thumbnailing style")
	}

	var thumbBuff bytes.Buffer
	if err := imaging.Encode(&thumbBuff, thumbImage, thumbFormat); err != nil {
		return nil, err
	}

	outputFile, err := context.CreateFileFromReader(thumbPath, &thumbBuff)
	if err != nil {
		return nil, err
	}
	outputFile.CopyProps(inputFile)
	return outputFile, nil
}
