package crawler

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/time/rate"

	"webcrawler/config"
	"webcrawler/storage"
	"webcrawler/types"
	"webcrawler/utils"
)

type Crawler struct {
	config    *config.Config
	visited   map[string]bool
	visitedMu sync.Mutex
	queue     chan string
	wg        sync.WaitGroup
	limiter   *rate.Limiter
}

func NewCrawler(cfg *config.Config) *Crawler {
	return &Crawler{
		config:  cfg,
		visited: make(map[string]bool),
		queue:   make(chan string, cfg.QueueSize),
		limiter: rate.NewLimiter(rate.Every(time.Second/time.Duration(cfg.RateLimit)), 1),
	}
}

func (c *Crawler) Start(urls []string) {
	// Initialize workers
	for i := 0; i < c.config.WorkerCount; i++ {
		go c.worker()
	}

	// Add initial URLs to queue
	for _, url := range urls {
		c.addToQueue(url)
	}

	// Wait for all tasks to complete
	c.wg.Wait()
	close(c.queue)

	fmt.Printf("\nCrawling complete. Results saved to results.json\n")
}

func (c *Crawler) worker() {
	for url := range c.queue {
		c.fetchURL(url)
	}
}

func (c *Crawler) fetchURL(targetURL string) {
	defer c.wg.Done()

	// Respect rate limiter
	c.limiter.Wait(context.Background())

	// Check robots.txt
	if !utils.CanCrawl(targetURL, c.config.UserAgent) {
		fmt.Println("\r[Blocked by robots.txt]", targetURL)
		return
	}

	// Check blacklist
	if utils.IsBlacklisted(targetURL) {
		fmt.Println("\r[Blacklisted]", targetURL)
		return
	}

	// Fetch and process the URL
	doc, err := c.fetch(targetURL)
	if err != nil {
		fmt.Println("\r[Failed]", targetURL, err)
		return
	}

	// Extract and save data
	data := c.extractData(targetURL, doc)
	storage.SaveData(data)

	// Queue new links
	c.queueNewLinks(targetURL, doc)

	fmt.Println("\r[Crawled]", targetURL)
}

func (c *Crawler) fetch(targetURL string) (*goquery.Document, error) {
	resp, err := http.Get(targetURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, err
	}
	defer resp.Body.Close()

	return goquery.NewDocumentFromReader(resp.Body)
}

func (c *Crawler) extractData(url string, doc *goquery.Document) types.PageData {
	return types.PageData{
		URL:   url,
		Title: doc.Find("title").Text(),
	}
}

func (c *Crawler) queueNewLinks(baseURL string, doc *goquery.Document) {
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		if link, exists := s.Attr("href"); exists {
			if absoluteURL := utils.ResolveURL(baseURL, link); absoluteURL != "" {
				c.addToQueue(absoluteURL)
				fmt.Println("\r[Queued]", absoluteURL)
			}
		}
	})
}

func (c *Crawler) addToQueue(targetURL string) {
	c.visitedMu.Lock()
	defer c.visitedMu.Unlock()

	if _, seen := c.visited[targetURL]; !seen && !utils.IsBlacklisted(targetURL) {
		c.visited[targetURL] = true
		c.wg.Add(1)
		c.queue <- targetURL
	}
}