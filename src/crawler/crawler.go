package crawler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
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
	visited   map[string]int // Track depth
	visitedMu sync.Mutex
	queue     chan string
	wg        sync.WaitGroup
	processed int64
	limiter   *rate.Limiter
}

func NewCrawler(cfg *config.Config) *Crawler {
	return &Crawler{
		config:  cfg,
		visited: make(map[string]int),
		queue:   make(chan string, cfg.QueueSize),
		limiter: rate.NewLimiter(rate.Every(time.Second/time.Duration(cfg.RateLimit)), 1),
	}
}

func (c *Crawler) Start(ctx context.Context, urls []string) {
	// Initialize workers
	for i := 0; i < c.config.WorkerCount; i++ {
		go c.worker(ctx)
	}

	// Add initial URLs to queue
	for _, url := range urls {
		c.addToQueue(url, 0) // Start with depth 0
	}

	// Wait for context cancellation or completion
	select {
	case <-ctx.Done():
		fmt.Println("\nCrawling interrupted.")
	case <-func() chan struct{} {
		done := make(chan struct{})
		go func() {
			c.wg.Wait()
			close(done)
		}()
		return done
	}():
		fmt.Println("\nCrawling complete. Results saved to results.json")
	}

	close(c.queue)
}

func (c *Crawler) worker(ctx context.Context) {
	for url := range c.queue {
		select {
		case <-ctx.Done():
			return
		default:
			c.fetchURL(url)
		}
	}
}

func (c *Crawler) fetchURL(targetURL string) {
	defer c.wg.Done()

	// Respect rate limiter
	c.limiter.Wait(context.Background())

	// Update progress metrics
	c.visitedMu.Lock()
	processed := c.processed
	c.visitedMu.Unlock()
	utils.UpdateProgress(int64(len(c.queue)), processed)

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
	depth := c.visited[targetURL]
	c.queueNewLinks(targetURL, doc, depth)

	fmt.Println("\r[Crawled]", targetURL)

	// Increment processed counter
	c.visitedMu.Lock()
	c.processed++
	c.visitedMu.Unlock()
}

func (c *Crawler) fetch(targetURL string) (*goquery.Document, error) {
	client := &http.Client{
		Timeout: 10 * time.Second, // Set a timeout
	}

	resp, err := client.Get(targetURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received HTTP status %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}

	// Parse Last-Modified header
	lastModified := resp.Header.Get("Last-Modified")
	lastModifiedTime, err := time.Parse(time.RFC1123, lastModified)
	if err != nil {
		lastModifiedTime = time.Now() // Fallback to current time
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	// Store LastModified in the document's context
	doc.Selection = doc.Selection.SetAttr("data-last-modified", lastModifiedTime.Format(time.RFC3339))

	return doc, nil
}

func (c *Crawler) extractData(url string, doc *goquery.Document) types.PageData {
	if doc == nil {
		return types.PageData{}
	}

	// Extract title
	title := doc.Find("title").Text()

	// Extract description (meta tag)
	description := ""
	doc.Find("meta[name='description']").Each(func(i int, s *goquery.Selection) {
		if desc, exists := s.Attr("content"); exists {
			description = desc
		}
	})

	// Extract meta tags
	var meta []types.Meta
	doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		if name, exists := s.Attr("name"); exists {
			if content, exists := s.Attr("content"); exists {
				meta = append(meta, types.Meta{Name: name, Content: content})
			}
		}
	})

	// Extract links
	var links []string
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		if link, exists := s.Attr("href"); exists {
			if absoluteURL := utils.ResolveURL(url, link); absoluteURL != "" {
				links = append(links, absoluteURL)
			}
		}
	})

	// Extract language
	language := ""
	doc.Find("html").Each(func(i int, s *goquery.Selection) {
		if lang, exists := s.Attr("lang"); exists {
			language = lang
		}
	})

	// Extract favicon
	favicon := ""
	doc.Find("link[rel='icon'], link[rel='shortcut icon']").Each(func(i int, s *goquery.Selection) {
		if icon, exists := s.Attr("href"); exists {
			favicon = utils.ResolveURL(url, icon)
		}
	})

	// Extract LastModified
	lastModified := doc.Selection.AttrOr("data-last-modified", time.Now().Format(time.RFC3339))
	lastModifiedTime, _ := time.Parse(time.RFC3339, lastModified)

	return types.PageData{
		URL:          url,
		Title:        title,
		Description:  description,
		Meta:         meta,
		LastModified: lastModifiedTime,
		// Links:        links,
		Language:     language,
		Favicon:      favicon,
	}
}

func (c *Crawler) queueNewLinks(baseURL string, doc *goquery.Document, depth int) {
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		if link, exists := s.Attr("href"); exists {
			if absoluteURL := utils.ResolveURL(baseURL, link); absoluteURL != "" {
				c.addToQueue(absoluteURL, depth+1)
				fmt.Println("\r[Queued]", absoluteURL)
			}
		}
	})
}

func (c *Crawler) addToQueue(targetURL string, depth int) {
	c.visitedMu.Lock()
	defer c.visitedMu.Unlock()

	if _, seen := c.visited[targetURL]; !seen && !utils.IsBlacklisted(targetURL) && depth < c.config.MaxDepth {
		c.visited[targetURL] = depth
		c.wg.Add(1)
		c.queue <- targetURL
		utils.UpdateProgress(int64(len(c.queue)), c.processed)
	}
}
