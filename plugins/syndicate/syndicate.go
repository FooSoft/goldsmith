package syndicate

import (
	"bytes"
	"fmt"
	"net/url"
	"sort"
	"sync"
	"time"

	"git.foosoft.net/alex/goldsmith"
	"github.com/gorilla/feeds"
)

type feed struct {
	config FeedConfig

	items itemList
	lock  sync.Mutex
}

type item struct {
	title       string
	authorName  string
	authorEmail string
	description string
	id          string
	updated     time.Time
	created     time.Time
	content     string

	url string
}

type itemList []item

func (self itemList) Len() int {
	return len(self)
}

func (self itemList) Less(i, j int) bool {
	if self[i].created != self[j].created {
		return self[i].created.Before(self[j].created)
	}

	if self[i].updated != self[j].updated {
		return self[i].updated.Before(self[j].updated)
	}

	if self[i].url != self[j].url {
		return self[i].url < self[j].url
	}

	return false
}

func (self itemList) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

type FeedConfig struct {
	AtomPath string
	JsonPath string
	RssPath  string

	Title       string
	Url         string
	Description string
	AuthorName  string
	AuthorEmail string
	Updated     time.Time
	Created     time.Time
	Id          string
	Subtitle    string
	Copyright   string
	ImageUrl    string
	ImageTitle  string
	ImageWidth  int
	ImageHeight int

	ItemConfig ItemConfig
}

type ItemConfig struct {
	TitleKey        string
	AuthorNameKey   string
	AuthorEmailKey  string
	DescriptionKey  string
	IdKey           string
	UpdatedKey      string
	CreatedKey      string
	ContentKey      string
	ContentFromFile bool
}

// Syndicate chainable context.
type Syndicate struct {
	baseUrl     string
	feedNameKey string

	feeds map[string]*feed
	lock  sync.Mutex
}

// New creates a new instance of the Syndicate plugin
func New(baseUrl, feedNameKey string) *Syndicate {
	return &Syndicate{
		baseUrl:     baseUrl,
		feedNameKey: feedNameKey,
		feeds:       make(map[string]*feed),
	}
}

func (*Syndicate) Name() string {
	return "syndicate"
}

func (self *Syndicate) WithFeed(name string, config FeedConfig) *Syndicate {
	self.feeds[name] = &feed{config: config}
	return self
}

func (self *Syndicate) Process(context *goldsmith.Context, inputFile *goldsmith.File) error {
	defer context.DispatchFile(inputFile)

	getString := func(key string) string {
		if len(key) == 0 {
			return ""
		}

		prop, ok := inputFile.Prop(key)
		if !ok {
			return ""
		}

		result, ok := prop.(string)
		if !ok {
			return ""
		}

		return result
	}

	getDate := func(key string) time.Time {
		if len(key) == 0 {
			return time.Time{}
		}

		prop, ok := inputFile.Prop(key)
		if !ok {
			return time.Time{}
		}

		result, ok := prop.(time.Time)
		if !ok {
			return time.Time{}
		}

		return result
	}

	feedName := getString(self.feedNameKey)
	if len(feedName) == 0 {
		return nil
	}

	self.lock.Lock()
	feed, ok := self.feeds[feedName]
	self.lock.Unlock()

	if !ok {
		return fmt.Errorf("feed %s has is not configured", feedName)
	}

	baseUrl, err := url.Parse(self.baseUrl)
	if err != nil {
		return err
	}

	currUrl, err := url.Parse(inputFile.Path())
	if err != nil {
		return err
	}

	item := item{
		title:       getString(feed.config.ItemConfig.TitleKey),
		authorName:  getString(feed.config.ItemConfig.AuthorNameKey),
		authorEmail: getString(feed.config.ItemConfig.AuthorEmailKey),
		description: getString(feed.config.ItemConfig.DescriptionKey),
		id:          getString(feed.config.ItemConfig.IdKey),
		updated:     getDate(feed.config.ItemConfig.UpdatedKey),
		created:     getDate(feed.config.ItemConfig.CreatedKey),
		content:     getString(feed.config.ItemConfig.ContentKey),
		url:         baseUrl.ResolveReference(currUrl).String(),
	}

	if len(item.content) == 0 && feed.config.ItemConfig.ContentFromFile {
		var buff bytes.Buffer
		if _, err := inputFile.WriteTo(&buff); err != nil {
			return err
		}

		item.content = string(buff.Bytes())
	}

	if len(item.id) == 0 {
		item.id = item.url
	}

	feed.lock.Lock()
	feed.items = append(feed.items, item)
	feed.lock.Unlock()

	return nil
}

func (self *feed) output(context *goldsmith.Context) error {
	feed := feeds.Feed{
		Title:       self.config.Title,
		Link:        &feeds.Link{Href: self.config.Url},
		Description: self.config.Description,
		Updated:     self.config.Updated,
		Created:     self.config.Created,
		Id:          self.config.Id,
		Subtitle:    self.config.Subtitle,
		Copyright:   self.config.Copyright,
	}

	if len(self.config.AuthorName) > 0 || len(self.config.AuthorEmail) > 0 {
		feed.Author = &feeds.Author{
			Name:  self.config.AuthorName,
			Email: self.config.AuthorEmail,
		}
	}

	if len(self.config.ImageUrl) > 0 {
		feed.Image = &feeds.Image{
			Url:    self.config.ImageUrl,
			Title:  self.config.ImageTitle,
			Width:  self.config.ImageWidth,
			Height: self.config.ImageHeight,
		}
	}

	sort.Sort(self.items)

	for _, item := range self.items {
		feedItem := feeds.Item{
			Title:       item.title,
			Description: item.description,
			Id:          item.id,
			Updated:     item.updated,
			Created:     item.created,
			Content:     item.content,
			Link:        &feeds.Link{Href: item.url},
		}

		if len(item.authorName) > 0 || len(item.authorEmail) > 0 {
			feedItem.Author = &feeds.Author{
				Name:  item.authorName,
				Email: item.authorEmail,
			}
		}

		feed.Items = append(feed.Items, &feedItem)
	}

	if len(self.config.AtomPath) > 0 {
		var buff bytes.Buffer
		if err := feed.WriteAtom(&buff); err != nil {
			return err
		}

		file, err := context.CreateFileFromReader(self.config.AtomPath, bytes.NewReader(buff.Bytes()))
		if err != nil {
			return err
		}

		context.DispatchFile(file)
	}

	if len(self.config.RssPath) > 0 {
		var buff bytes.Buffer
		if err := feed.WriteRss(&buff); err != nil {
			return err
		}

		file, err := context.CreateFileFromReader(self.config.RssPath, bytes.NewReader(buff.Bytes()))
		if err != nil {
			return err
		}

		context.DispatchFile(file)
	}

	if len(self.config.JsonPath) > 0 {
		var buff bytes.Buffer
		if err := feed.WriteJSON(&buff); err != nil {
			return err
		}

		file, err := context.CreateFileFromReader(self.config.JsonPath, bytes.NewReader(buff.Bytes()))
		if err != nil {
			return err
		}

		context.DispatchFile(file)
	}

	return nil
}

func (self *Syndicate) Finalize(context *goldsmith.Context) error {
	for _, feed := range self.feeds {
		feed.output(context)
	}

	return nil
}
