package whisper_test

import (
	"fmt"
	"log"

	"github.com/kawai-network/whisper"
)

func ExampleLibraryDownloader_DownloadLatest() {
	// Create a downloader that saves to "./libs" directory
	downloader := whisper.NewLibraryDownloader("./libs")

	// Download the latest library for current platform
	path, err := downloader.DownloadLatest()
	if err != nil {
		log.Fatalf("Failed to download library: %v", err)
	}

	fmt.Printf("Library downloaded to: %s\n", path)

	// Now you can use the downloaded library
	w, err := whisper.New("./libs")
	if err != nil {
		log.Fatalf("Failed to initialize whisper: %v", err)
	}
	defer w.Close()

	// Use whisper...
	_ = w
}

func ExampleLibraryDownloader() {
	// Detect current platform
	platform := whisper.DetectPlatform()
	fmt.Printf("Platform: %s/%s\n", platform.OS, platform.Arch)
	fmt.Printf("Library extension: %s\n", platform.Extension)

	// Create downloader
	downloader := whisper.NewLibraryDownloader("./libs")

	// Get latest release info
	release, err := downloader.GetLatestRelease()
	if err != nil {
		log.Fatalf("Failed to get latest release: %v", err)
	}

	fmt.Printf("Latest release: %s\n", release.TagName)

	// Select best library for platform
	asset, err := downloader.SelectBestLibrary(release, platform)
	if err != nil {
		log.Fatalf("No suitable library found: %v", err)
	}

	fmt.Printf("Selected: %s (%s variant)\n", asset.Name, asset.Variant)

	// Download with resume support
	path, err := downloader.Download(asset)
	if err != nil {
		log.Fatalf("Download failed: %v", err)
	}

	fmt.Printf("Downloaded to: %s\n", path)
}
