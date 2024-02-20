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

	"git.foosoft.net/alex/goldsmith"
	"git.foosoft.net/alex/goldsmith/filters/wildcard"
	"github.com/disintegration/imaging"
)

// Namer callback function builds thumbnail file paths based on the original file path.
// An empty path can be returned if a thumbnail should not be generated for the current file.
type Namer func(string, int) string

// Desired thumbnailing styles.
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
		128,
		Fit,
		color.Transparent,
		namer,
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

func (*Thumbnail) Initialize(context *goldsmith.Context) error {
	context.Filter(wildcard.New("**/*.jpg", "**/*.jpeg", "**/*.gif", "**/*.png"))
	return nil
}

func (self *Thumbnail) Process(context *goldsmith.Context, inputFile goldsmith.File) error {
	defer context.DispatchFile(inputFile)

	thumbPath := self.namer(inputFile.Path(), self.size)
	if len(thumbPath) == 0 {
		return nil
	}

	if outputFile := context.RetrieveCachedFile(thumbPath, inputFile); outputFile != nil {
		outputFile.CopyProps(inputFile)
		context.DispatchFile(outputFile)
		return nil
	}

	outputFile, err := self.thumbnail(context, inputFile, thumbPath)
	if err != nil {
		return err
	}

	context.DispatchAndCacheFile(outputFile, inputFile)
	return nil
}

func (self *Thumbnail) thumbnail(context *goldsmith.Context, inputFile goldsmith.File, thumbPath string) (goldsmith.File, error) {
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

	return context.CreateFileFromReader(thumbPath, &thumbBuff)
}
