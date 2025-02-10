package main

import (
	"fmt"
	"time"

	"webcrawler/config"
	"webcrawler/crawler"
	"webcrawler/utils"
)

func main() {
	startURLs, err := utils.ReadURLs("urls.txt")
	if err != nil {
		fmt.Println(err)
		return
	}

	err = utils.ReadBlacklist("blacklist.txt")
	if err != nil {
		fmt.Println(err)
		return
	}

	start := time.Now()
	go utils.DisplayProgress(start)

	crawler := crawler.NewCrawler(config.DefaultConfig())
	crawler.Start(startURLs)
}