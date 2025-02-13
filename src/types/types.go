package types

import "time"

type PageData struct {
	URL       	string    `json:"url"`            // Page URL
	Title     	string    `json:"title"`          // Page title
	Description string    `json:"description"`    // Page description
	Meta      	[]Meta    `json:"meta"`           // Page metadata
	LastModified time.Time `json:"last_modified"` // Page last modified time
	// Links     	[]string  `json:"links"`          // Page links
	Language  	string    `json:"language"`       // Page language
	Favicon   	string    `json:"favicon"`        // Page favicon
}

type Meta struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}


