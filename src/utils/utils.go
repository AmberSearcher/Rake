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

var (
	robotsMap = make(map[string]*robotstxt.RobotsData)
	robotsMu  sync.Mutex
	blacklist = make(map[string]bool)
)

func ReadURLs(filename string) ([]string, error) {
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

func ReadBlacklist(filename string) error {
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

func IsBlacklisted(targetURL string) bool {
	for pattern := range blacklist {
		if strings.Contains(targetURL, pattern) {
			return true
		}
	}
	return false
}

func CanCrawl(targetURL, userAgent string) bool {
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
		fmt.Printf("\rCurrently scraping: Website #%d Running Time: %.2fs", 
			len(blacklist), // This should be replaced with actual queue length
			time.Since(start).Seconds())
	}
}