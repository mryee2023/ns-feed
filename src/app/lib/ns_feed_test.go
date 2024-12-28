package lib

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mmcdole/gofeed"
)

func TestLinuxDoFeed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	fp := gofeed.NewParser()
	feed, err := fp.ParseURLWithContext("https://linux.do/latest.rss", ctx)
	if err != nil {
		fmt.Println(err)
	}
	if feed == nil {
		fmt.Println("feed is nil")
		return
	}

	for _, item := range feed.Items {
		fmt.Println(item.Title, " ", item.PublishedParsed.Add(time.Hour*8).Format("2006-01-02 15:04:05"), " ", item.Link)
	}
}
