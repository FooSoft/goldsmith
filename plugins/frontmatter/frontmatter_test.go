package frontmatter

import (
	"io"
	"strings"
	"testing"

	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/harness"
	"go.foosoft.net/goldsmith/plugins/layout"
)

func Test(self *testing.T) {
	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.
				Chain(New()).
				Chain(layout.New())
		},
	)
}

func TestParsePreservesBodyExactly(t *testing.T) {
	for _, input := range []string{
		"body without a newline",
		"first\r\nsecond\r\n",
		"---\r\nTitle: test\r\n---\r\nbody without a newline",
	} {
		_, body, err := parse(strings.NewReader(input))
		if err != nil {
			t.Fatal(err)
		}

		expected := input
		if strings.HasPrefix(input, "---") {
			expected = "body without a newline"
		}
		actual, err := io.ReadAll(body)
		if err != nil {
			t.Fatal(err)
		}
		if string(actual) != expected {
			t.Fatalf("body mismatch: got %q, want %q", actual, expected)
		}
	}
}

func TestParseSupportsLongLines(t *testing.T) {
	expected := strings.Repeat("x", 128*1024)
	_, body, err := parse(strings.NewReader(expected))
	if err != nil {
		t.Fatal(err)
	}
	if body.String() != expected {
		t.Fatalf("got a body of %d bytes, want %d", body.Len(), len(expected))
	}
}

func TestParseSupportsNestedJSON(t *testing.T) {
	input := "{\n\"Nested\": {\n\"Values\": [1, 2]\n}\n}\nbody"
	meta, body, err := parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	nested, ok := meta["Nested"].(map[string]interface{})
	if !ok {
		t.Fatalf("nested metadata has type %T", meta["Nested"])
	}
	values, ok := nested["Values"].([]interface{})
	if !ok || len(values) != 2 {
		t.Fatalf("nested values were not parsed: %#v", nested["Values"])
	}
	if body.String() != "body" {
		t.Fatalf("got body %q, want %q", body.String(), "body")
	}
}
