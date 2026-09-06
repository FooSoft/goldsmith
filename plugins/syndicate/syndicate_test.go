package syndicate

import (
	"strings"
	"testing"

	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/harness"
	"go.foosoft.net/goldsmith/plugins/frontmatter"
)

func Test(self *testing.T) {
	feedConfig := FeedConfig{
		Title:       "Feed Title",
		Url:         "https://foosoft.net",
		Description: "Feed Description",
		AuthorName:  "Author Name",
		AuthorEmail: "Author Email",
		Id:          "Feed Id",
		Subtitle:    "Feed Subtitle",
		Copyright:   "Feed Copyright",
		ImageUrl:    "Feed Image Url",
		AtomPath:    "feed.atom",
		RssPath:     "feed.xml",
		JsonPath:    "feed.json",
	}

	feedConfig.ItemConfig = ItemConfig{
		TitleKey:        "Title",
		AuthorNameKey:   "AuthorName",
		AuthorEmailKey:  "AuthorEmail",
		DescriptionKey:  "Description",
		IdKey:           "Id",
		UpdatedKey:      "Updated",
		CreatedKey:      "Created",
		ContentKey:      "Content",
		ContentFromFile: true,
	}
	plugin := New("https://foosoft.net", "FeedName").WithFeed("posts", feedConfig)

	harness.Validate(
		self,
		func(gs *goldsmith.Goldsmith) {
			gs.
				Chain(frontmatter.New()).
				Chain(plugin)
		},
	)
}

func TestFinalizeReportsOutputErrors(t *testing.T) {
	plugin := New("https://example.com", "Feed").WithFeed("invalid", FeedConfig{
		AtomPath: "../outside.atom",
	})

	err := plugin.Finalize(new(goldsmith.Context))
	if err == nil || !strings.Contains(err.Error(), "escapes its root") {
		t.Fatalf("expected an unsafe feed path error, got %v", err)
	}
}
