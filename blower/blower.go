package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/vmihailenco/msgpack/v5"
)

var (
	inputDir    = "./data/"         // Directory containing .awf files
	outputFile  = "combined_data.awf"
	outputTxt   = "links.txt"
	fileMutex   sync.Mutex
	seenURLs    = make(map[string]bool) // To remove duplicates
	uniquePages []PageData        // Store unique entries
)

type PageData struct {
	URL   string `json:"url"` // Page URL
	Title string `json:"title"` // Page title
}

// Read an AWF file and extract PageData
func readAWFFile(filename string) error {
	fmt.Println("Processing:", filename)

	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	for {
		var length uint64
		err := binary.Read(reader, binary.LittleEndian, &length)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading length: %v", err)
		}

		// Sanity check for the length
		const maxLength = 10 * 1024 * 1024 // 10MB
		if length > maxLength {
			return fmt.Errorf("invalid length %d in file %s: exceeds maximum allowed size", length, filename)
		}

		data := make([]byte, length)
		_, err = io.ReadFull(reader, data)
		if err != nil {
			return fmt.Errorf("error reading data: %v", err)
		}

		var page PageData
		err = msgpack.Unmarshal(data, &page)
		if err != nil {
			fmt.Printf("Error decoding record in %s: %v\n", filename, err)
			continue
		}

		if _, exists := seenURLs[page.URL]; !exists {
			seenURLs[page.URL] = true
			uniquePages = append(uniquePages, page)
		}
	}

	return nil
}

// Write unique data back to AWF format
func writeCombinedAWF() error {
	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	for _, page := range uniquePages {
		binData, err := msgpack.Marshal(page)
		if err != nil {
			fmt.Println("Error encoding:", err)
			continue
		}

		// Write length prefix
		if err := binary.Write(writer, binary.LittleEndian, uint64(len(binData))); err != nil {
			return err
		}

		// Write data
		_, err = writer.Write(binData)
		if err != nil {
			return err
		}
	}

	fmt.Println("Combined data saved to", outputFile)
	return nil
}

// Write links to a text file
func writeLinksTxt() error {
	file, err := os.Create(outputTxt)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	for _, page := range uniquePages {
		_, err := writer.WriteString(page.URL + "\n")
		if err != nil {
			return err
		}
	}

	fmt.Println("Links saved to", outputTxt)
	return nil
}

func main() {
	files, err := filepath.Glob(filepath.Join(inputDir, "*.awf"))
	if err != nil || len(files) == 0 {
		fmt.Println("No .awf files found in", inputDir)
		return
	}

	for _, file := range files {
		err := readAWFFile(file)
		if err != nil {
			fmt.Printf("Error processing %s: %v\n", file, err)
			return // Exit early if a file is corrupted
		}
	}

	if err := writeCombinedAWF(); err != nil {
		fmt.Println("Error saving combined AWF:", err)
	}

	if err := writeLinksTxt(); err != nil {
		fmt.Println("Error saving links:", err)
	}
}