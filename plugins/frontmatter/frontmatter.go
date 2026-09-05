// Package frontmatter extracts the metadata stored in your files. This
// metadata can include any information you want, but typically contains the
// page title, creation date, tags, layout template, and more. There are no
// requirements about what fields must be present; this is entirely up to you.
//
//	+++
//	Title = "My homepage"
//	Tags = ["best", "page", "ever"]
//	+++
//
// Metadata in YAML format is enclosed by three minus (-) characters:
//
//	---
//	Title: "My homepage"
//	Tags:
//	  - "best"
//	  - "page"
//	  - "ever"
//	---
//
// Metadata in JSON format is enclosed by brace characters ({ and }):
//
//	{
//	    "Title": "My homepage",
//	    "Tags": ["best", "page", "ever"]
//	}
//
// It is possible to enclose the frontmatter starting and ending delimiters in HTML
// comments. The comment has to start and stop on the same line as the delimiter.
//
//	<!-- ---
//	Title: "My homepage"
//	Tags:
//	  - "best"
//	  - "page"
//	  - "ever"
//	--- -->
//
// Normal page content immediately follows the metadata section. The metadata
// section is stripped after processed by this plugin.
package frontmatter

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/BurntSushi/toml"
	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/filters/wildcard"
	"gopkg.in/yaml.v2"
)

// Frontmatter chainable plugin context.
type FrontMatter struct{}

// New creates a new instance of the Frontmatter plugin.
func New() *FrontMatter {
	return new(FrontMatter)
}

func (*FrontMatter) Name() string {
	return "frontmatter"
}

func (*FrontMatter) Initialize(context *goldsmith.Context) error {
	context.Filter(wildcard.New("**/*.md", "**/*.markdown", "**/*.rst", "**/*.txt", "**/*.html", "**/*.htm"))
	return nil
}

func (*FrontMatter) Process(context *goldsmith.Context, inputFile *goldsmith.File) error {
	meta, body, err := parse(inputFile)
	if err != nil {
		return err
	}

	outputFile, err := context.CreateFileFromReader(inputFile.Path(), body)
	if err != nil {
		return err
	}

	outputFile.CopyProps(inputFile)
	for name, value := range meta {
		outputFile.SetProp(name, value)
	}

	context.DispatchFile(outputFile)
	return nil
}

func parse(reader io.Reader) (map[string]interface{}, *bytes.Buffer, error) {
	const (
		yamlOpener     = "---"
		yamlCloser     = "---"
		tomlOpener     = "+++"
		tomlCloser     = "+++"
		jsonOpener     = "{"
		jsonCloser     = "}"
		yamlOpenerHtml = "<!-- ---"
		yamlCloserHtml = "--- -->"
		tomlOpenerHtml = "<!-- +++"
		tomlCloserHtml = "+++ -->"
		jsonOpenerHtml = "<!-- {"
		jsonCloserHtml = "} -->"
	)

	meta := make(map[string]interface{})
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, nil, err
	}

	firstLine, offset := readLine(data, 0)
	var closer string
	switch strings.TrimSpace(string(firstLine)) {
	case tomlOpener:
		closer = tomlCloser
	case yamlOpener:
		closer = yamlCloser
	case jsonOpener:
		closer = jsonCloser
	case tomlOpenerHtml:
		closer = tomlCloserHtml
	case yamlOpenerHtml:
		closer = yamlCloserHtml
	case jsonOpenerHtml:
		closer = jsonCloserHtml
	default:
		return meta, bytes.NewBuffer(data), nil
	}

	var front bytes.Buffer
	if closer == jsonCloser || closer == jsonCloserHtml {
		front.WriteString(jsonOpener)
	}

	for {
		if offset >= len(data) {
			return nil, nil, errors.New("unterminated front matter block")
		}

		line, next := readLine(data, offset)
		if strings.TrimSpace(string(line)) == closer {
			if closer == jsonCloser || closer == jsonCloserHtml {
				front.WriteString(jsonCloser)
			}
			offset = next
			break
		}

		front.Write(line)
		offset = next
	}

	switch closer {
	case tomlCloser, tomlCloserHtml:
		if err := toml.Unmarshal(front.Bytes(), &meta); err != nil {
			return nil, nil, err
		}
	case yamlCloser, yamlCloserHtml:
		if err := yaml.Unmarshal(front.Bytes(), &meta); err != nil {
			return nil, nil, err
		}
	case jsonCloser, jsonCloserHtml:
		if err := json.Unmarshal(front.Bytes(), &meta); err != nil {
			return nil, nil, err
		}
	}

	return meta, bytes.NewBuffer(data[offset:]), nil
}

func readLine(data []byte, offset int) ([]byte, int) {
	if offset >= len(data) {
		return nil, len(data)
	}

	if end := bytes.IndexByte(data[offset:], '\n'); end >= 0 {
		end += offset + 1
		return data[offset:end], end
	}

	return data[offset:], len(data)
}
