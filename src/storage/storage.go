package storage

import (
	"encoding/json"
	"fmt"
	"os"

	"webcrawler/types"
)

func SaveData(data types.PageData) {
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