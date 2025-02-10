package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/temoto/robotstxt"
	"golang.org/x/time/rate"
)

const (
	WorkerCount = 10 // Number of concurrent workers
	RateLimit   = 5 // Requests per second per domain
)

var (
	visited   = make(map[string]bool) // Tracks visited URLs
	visitedMu sync.Mutex              // Mutex for safe concurrent access
	queue     = make(chan string, 100000) // Channel-based queue
	wg        sync.WaitGroup           // WaitGroup to sync goroutines
	limiter   = rate.NewLimiter(rate.Every(time.Second/RateLimit), 1) // Rate limiter
	robotsMap = make(map[string]*robotstxt.RobotsData) // Robots.txt cache
	robotsMu  sync.Mutex
	blacklist = make(map[string]bool) // Blacklisted URL patterns
)

// Holds the crawled data
type PageData struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// Fetch robots.txt and check if we are allowed
func canCrawl(targetURL string) bool {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return false
	}
	domain := parsedURL.Host

	robotsMu.Lock()
	robots, exists := robotsMap[domain]
	robotsMu.Unlock()

	if !exists {
		robotsURL := parsedURL.Scheme + "://" + domain + "/robots.txt"
		resp, err := http.Get(robotsURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			robots, err = robotstxt.FromBytes(body)
			if err == nil {
				robotsMu.Lock()
				robotsMap[domain] = robots
				robotsMu.Unlock()
			}
		}
	}

	if robots != nil {
		return robots.TestAgent(parsedURL.Path, "AmberRake")
	}
	return true
}

// Check if URL is blacklisted
func isBlacklisted(targetURL string) bool {
	for pattern := range blacklist {
		if strings.Contains(targetURL, pattern) {
			return true
		}
	}
	return false
}

// Fetch and process a URL
func fetchURL(targetURL string) {
	defer wg.Done()

	// Respect rate limiter
	limiter.Wait(context.Background())

	// Check robots.txt
	if !canCrawl(targetURL) {
		fmt.Println("\r[Blocked by robots.txt]", targetURL)
		return
	}

	// Check blacklist
	if isBlacklisted(targetURL) {
		fmt.Println("\r[Blacklisted]", targetURL)
		return
	}

	// Fetch the URL
	resp, err := http.Get(targetURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Println("\r[Failed]", targetURL, err)
		return
	}
	defer resp.Body.Close()

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		fmt.Println("\r[Parse Error]", targetURL, err)
		return
	}

	// Extract title
	title := strings.TrimSpace(doc.Find("title").Text())

	// Save data
	data := PageData{
		URL:   targetURL,
		Title: title,
	}
	saveData(data)

	// Extract and queue new links
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		link, exists := s.Attr("href")
		if exists {
			absoluteURL := resolveURL(targetURL, link)
			if absoluteURL != "" {
				addToQueue(absoluteURL)
				fmt.Println("\r[Queued]", absoluteURL)
			}
		}
	})

	fmt.Println("\r[Crawled]", targetURL)
}

// Resolve relative URLs
func resolveURL(base, link string) string {
	parsedBase, err := url.Parse(base)
	if err != nil {
		return ""
	}
	parsedLink, err := url.Parse(link)
	if err != nil {
		return ""
	}

	// Absolute and relative links
	if parsedLink.IsAbs() {
		return parsedLink.String()
	}
	return parsedBase.ResolveReference(parsedLink).String()
}

// Save to a JSON file
func saveData(data PageData) {
	file, err := os.OpenFile("results.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("\r[File Error]", err)
		return
	}
	defer file.Close()

	jsonData, _ := json.Marshal(data)
	file.WriteString(string(jsonData) + "\n")

	fmt.Println("\r[Saved]", data.URL)
}

// Add URL to the queue if not already visited
func addToQueue(targetURL string) {
	visitedMu.Lock()
	defer visitedMu.Unlock()

	// Normalize URL
	targetURL = strings.TrimRight(targetURL, "/")
	if _, seen := visited[targetURL]; !seen && !isBlacklisted(targetURL) {
		visited[targetURL] = true
		wg.Add(1)
		queue <- targetURL
	}
}

// Worker function to process URLs
func worker() {
	for targetURL := range queue {
		fetchURL(targetURL)
	}
}

// Read URLs 
func readURLs(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		urls = append(urls, scanner.Text())
	}
	return urls, scanner.Err()
}

// Read blacklist 
func readBlacklist(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		blacklist[scanner.Text()] = true
	}
	return scanner.Err()
}

// Start crawler with worker pool
func startCrawler(urls []string) {
	// Initialize workers
	for i := 0; i < WorkerCount; i++ {
		go worker()
	}

	// Add initial URLs to queue
	for _, url := range urls {
		addToQueue(url)
	}

	// Wait for all tasks to complete
	wg.Wait()
	close(queue)

	// Print final statistics
	fmt.Printf("Crawling complete. Results saved to results.json\n")
}

func main() {
	startURLs, err := readURLs("urls.txt")
	if err != nil {
		fmt.Println(err)
		return
	}

	err = readBlacklist("blacklist.txt")
	if err != nil {
		fmt.Println(err)
		return
	}
	start := time.Now()
	go func() {
		for {
			time.Sleep(time.Second / 8)

			// Explanation: This is the status bar at the bottom, the currently scrapping is the current # in the queue that is being worked on, the Queued is the total number of URLs that have been queued, and the Running Time is how long it has been running.
			fmt.Printf("\rCurrently scraping: Website #%d Total added: %d Running Time: %.2fs", len(queue), len(visited), time.Since(start).Seconds()) 
		}
	}()

	startCrawler(startURLs)
}