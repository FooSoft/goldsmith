package syndicate

import (
	"strings"
	"testing"
	"time"

	"go.foosoft.net/goldsmith"
	"go.foosoft.net/goldsmith/harness"
	"go.foosoft.net/goldsmith/plugins/frontmatter"
)

func TestItemOrderUsesInstantsInsteadOfTimeZones(t *testing.T) {
	created := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	updated := created.Add(time.Hour)
	zoned := created.In(time.FixedZone("offset", 3600))
	items := itemList{
		{created: created, updated: updated, url: "a"},
		{created: zoned, updated: updated.In(time.FixedZone("other", -3600)), url: "b"},
	}
	if !items.Less(0, 1) || items.Less(1, 0) {
		t.Fatal("equivalent instants did not fall back to URL order")
	}
	items[0].updated = updated.Add(-time.Minute)
	items[0].url = "z"
	if !items.Less(0, 1) {
		t.Fatal("update time did not break creation time tie")
	}
}

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
