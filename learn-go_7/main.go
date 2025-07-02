package main

import (
	"fmt"

	"github.com/mmcdole/gofeed"
)

func main() {
	fp := gofeed.NewParser()

	feed, err := fp.ParseURL("http://feeds.bbci.co.uk/news/world/rss.xml")
	if err != nil {
		panic(err)
	}

	for _, item := range feed.Items {
		fmt.Println("===================================")
		fmt.Println("Title:", item.Title)
		fmt.Println("Link:", item.Link)
		fmt.Println("Published:", item.Published)
		fmt.Println("Description:", item.Description)
		fmt.Println()
	}
}