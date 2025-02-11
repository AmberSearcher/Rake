package utils

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/temoto/robotstxt"
)

// Progress tracking
var (
	queueSize     int64
	processedSize int64
	progressMu    sync.Mutex
)

// UpdateProgress updates the crawler's progress metrics
func UpdateProgress(inQueue, processed int64) {
	progressMu.Lock()
	queueSize = inQueue
	processedSize = processed
	progressMu.Unlock()
}

var (
	robotsMap  = make(map[string]*robotstxt.RobotsData)
	robotsMu   sync.Mutex
	blacklist  = make(map[string]bool)
	bypassList = make(map[string]bool)
)

func ReadConfig(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)
	section := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		if strings.HasSuffix(line, ":") {
			section = strings.TrimSuffix(line, ":")
			continue
		}
		switch section {
		case "Websites":
			urls = append(urls, strings.Fields(line)...)
		case "Blacklist":
			for _, item := range strings.Fields(line) {
				blacklist[item] = true
			}
		case "Bypass":
			for _, item := range strings.Fields(line) {
				bypassList[item] = true
			}
		}
	}
	return urls, scanner.Err()
}

func IsBlacklisted(targetURL string) bool {
	for pattern := range blacklist {
		if strings.Contains(targetURL, pattern) {
			return true
		}
	}
	return false
}

func CanCrawl(targetURL, userAgent string) bool {
	if strings.Contains(targetURL, "://"+domainForBypass()) {
		return true
	}

	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return false
	}
	domain := parsedURL.Host

	robotsMu.Lock()
	robots, exists := robotsMap[domain]
	robotsMu.Unlock()

	if !exists {
		robots = fetchRobotsTxt(parsedURL, domain)
	}

	if robots != nil {
		return robots.TestAgent(parsedURL.Path, userAgent)
	}
	return true
}

func domainForBypass() string {
	var domain string
	for domain = range bypassList {
		return domain
	}
	return ""
}

func fetchRobotsTxt(parsedURL *url.URL, domain string) *robotstxt.RobotsData {
	robotsURL := parsedURL.Scheme + "://" + domain + "/robots.txt"
	resp, err := http.Get(robotsURL)
	if err == nil && resp.StatusCode == http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		robots, err := robotstxt.FromBytes(body)
		if err == nil {
			robotsMu.Lock()
			robotsMap[domain] = robots
			robotsMu.Unlock()
			return robots
		}
	}
	return nil
}

func ResolveURL(base, link string) string {
	parsedBase, err := url.Parse(base)
	if err != nil {
		return ""
	}
	parsedLink, err := url.Parse(link)
	if err != nil {
		return ""
	}

	if parsedLink.IsAbs() {
		return parsedLink.String()
	}
	return parsedBase.ResolveReference(parsedLink).String()
}

func DisplayProgress(start time.Time) {
	for {
		time.Sleep(time.Second / 8)
		progressMu.Lock()
		queue := queueSize
		processed := processedSize
		progressMu.Unlock()
		
		fmt.Printf("\rItems left in queue: %d, Items processed so far: %d, Running Time: %.2fs", 
			queue,
			processed,
			time.Since(start).Seconds())
	}
}
