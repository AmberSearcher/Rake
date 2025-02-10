package storage

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
	"sync"

	"webcrawler/types"

	"github.com/vmihailenco/msgpack/v5"
)

var (
	fileMutex   sync.Mutex
	fileHandle  *os.File
	writer      *bufio.Writer
	storageFile = "crawl_data.awf" // Default filename
)

// SetStorageFile configures output filename (thread-safe)
func SetStorageFile(filename string) {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	if fileHandle != nil {
		writer.Flush()
		fileHandle.Close()
		fileHandle = nil
	}
	storageFile = filename
}

func initStorage() error {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	if fileHandle == nil {
		f, err := os.OpenFile(storageFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		fileHandle = f
		writer = bufio.NewWriter(f)
	}
	return nil
}

func SaveData(data types.PageData) {
	if fileHandle == nil {
		if err := initStorage(); err != nil {
			fmt.Println("\r[File Error]", err)
			return
		}
	}

	// MessagePack serialization
	binData, err := msgpack.Marshal(data)
	if err != nil {
		fmt.Println("\r[Serialization Error]", err)
		return
	}

	// Write with length prefix for C++ decoding
	if err := writeRecord(binData); err != nil {
		fmt.Println("\r[Write Error]", err)
		return
	}

	fmt.Println("\r[Saved]", data.URL)
}

// C++ compatible binary format:
// [8-byte little-endian length][msgpack data]
func writeRecord(data []byte) error {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	// Write length header
	if err := binary.Write(writer, binary.LittleEndian, uint64(len(data))); err != nil {
		return err
	}

	// Write messagepack payload
	if _, err := writer.Write(data); err != nil {
		return err
	}

	// Flush every 100 writes
	if writer.Buffered() > 100*1024 { // 100KB buffer
		return writer.Flush()
	}
	return nil
}

func Close() {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	if writer != nil {
		writer.Flush()
	}
	if fileHandle != nil {
		fileHandle.Close()
	}
}
