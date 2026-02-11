//go:build download
// +build download

package main

import (
	"log"
	"os"

	"github.com/kawai-network/whisper"
)

func main() {
	downloader := whisper.NewLibraryDownloader(".")

	path, err := downloader.DownloadLatest()
	if err != nil {
		log.Fatalf("Failed to download library: %v", err)
	}

	log.Printf("Library downloaded to: %s", path)

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Fatalf("Downloaded file not found: %s", path)
	}

	log.Println("Library download successful")
}
