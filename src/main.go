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
	// Read configuration
	startURLs, err := utils.ReadConfig("config.rcf")
	if err != nil {
		fmt.Println(err)
		return
	}

	// Welcome message with config info
	fmt.Println(`

 _____       _        
|  __ \     | |       
| |__) |__ _| | _____ 
|  _  // _ \| |/ / _ \
| | \ \ (_| |   <  __/
|_|  \_\__,_|_|\_\___|
	

Welcome to Rake, the web crawler! 
	`)
	fmt.Printf("Loaded configuration: %d workers, rate limit: %d requests/sec\n", config.DefaultConfig().WorkerCount, config.DefaultConfig().RateLimit)
	fmt.Printf("Starting URLs: %v\n", startURLs)

	// Wait 2 seconds before starting the crawler
	time.Sleep(2 * time.Second)
	
	// Display progress
	start := time.Now()
	go utils.DisplayProgress(start)

	// Initialize storage
	storage.Init()

	// Start the crawler
	crawler := crawler.NewCrawler(config.DefaultConfig())
	crawler.Start(startURLs)
}


