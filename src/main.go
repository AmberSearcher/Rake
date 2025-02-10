package main

import (
	"fmt"
	"time"

	"webcrawler/config"
	"webcrawler/crawler"
	"webcrawler/storage"
	"webcrawler/utils"
)

func main() {
	// Read URLs
	startURLs, err := utils.ReadURLs("urls.txt")
	if err != nil {
		fmt.Println(err)
		return
	}

	// Read blacklist
	err = utils.ReadBlacklist("blacklist.txt")
	if err != nil {
		fmt.Println(err)
		return
	}

	// Display progress
	start := time.Now()
	go utils.DisplayProgress(start)

	// Initialize storage
	storage.Init()

	// Start the crawler
	crawler := crawler.NewCrawler(config.DefaultConfig())
	crawler.Start(startURLs)
}
