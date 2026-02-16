package whisper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/kawai-network/grab"
)

const (
	githubAPIURL    = "https://api.github.com/repos/kawai-network/whisper/releases/latest"
	downloadTimeout = 300 * time.Second
)

// PlatformInfo holds platform-specific information
type PlatformInfo struct {
	OS             string
	Arch           string
	Extension      string
	Prefix         string
	SupportsAVX    bool
	SupportsAVX2   bool
	SupportsAVX512 bool
}

// LibraryDownloader handles downloading platform-specific libraries
type LibraryDownloader struct {
	client    *grab.Client
	targetDir string
}

// NewLibraryDownloader creates a new library downloader
func NewLibraryDownloader(targetDir string) *LibraryDownloader {
	return &LibraryDownloader{
		client:    grab.NewClient(),
		targetDir: targetDir,
	}
}

// DetectPlatform detects the current platform and returns library info
func DetectPlatform() *PlatformInfo {
	info := &PlatformInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	switch runtime.GOOS {
	case "darwin":
		info.Extension = ".dylib"
		info.Prefix = "lib"
		info.SupportsAVX = false
		info.SupportsAVX2 = false
		info.SupportsAVX512 = false
	case "windows":
		info.Extension = ".dll"
		info.Prefix = ""
		info.SupportsAVX = false
		info.SupportsAVX2 = false
		info.SupportsAVX512 = false
	default: // Linux
		info.Extension = ".so"
		info.Prefix = "lib"
		// Check CPU features on Linux
		info.SupportsAVX = true // Will be refined with cpuid check
		info.SupportsAVX2 = true
		info.SupportsAVX512 = true
	}

	return info
}

// LibraryName returns the platform-specific library name for the given OS
func LibraryName(goos string) string {
	var prefix, extension string
	switch goos {
	case "darwin":
		prefix = "lib"
		extension = ".dylib"
	case "windows":
		prefix = ""
		extension = ".dll"
	default: // Linux
		prefix = "lib"
		extension = ".so"
	}
	return prefix + "gowhisper" + extension
}

// GetLatestRelease fetches the latest release info from GitHub
func (d *LibraryDownloader) GetLatestRelease() (*ReleaseInfo, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(githubAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release info: %w", err)
	}

	return &release, nil
}

// ReleaseInfo represents GitHub release information
type ReleaseInfo struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               int64  `json:"size"`
	} `json:"assets"`
}

// SelectBestLibrary selects the best library file for the platform
func (d *LibraryDownloader) SelectBestLibrary(release *ReleaseInfo, platform *PlatformInfo) (*LibraryAsset, error) {
	var candidates []LibraryAsset

	for _, asset := range release.Assets {
		// Check if asset matches platform
		if !d.matchesPlatform(asset.Name, platform) {
			continue
		}

		// Determine variant (fallback, avx, avx2, avx512)
		variant := d.detectVariant(asset.Name)

		candidates = append(candidates, LibraryAsset{
			Name:     asset.Name,
			URL:      asset.BrowserDownloadURL,
			Size:     asset.Size,
			Variant:  variant,
			Platform: platform,
		})
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no suitable library found for platform %s/%s", platform.OS, platform.Arch)
	}

	// Select best variant based on platform capabilities
	return d.selectBestVariant(candidates, platform), nil
}

// LibraryAsset represents a downloadable library
type LibraryAsset struct {
	Name     string
	URL      string
	Size     int64
	Variant  string
	Platform *PlatformInfo
}

func (d *LibraryDownloader) matchesPlatform(filename string, platform *PlatformInfo) bool {
	expectedName := platform.Prefix + "gowhisper"

	// Check for platform-specific extensions
	switch platform.OS {
	case "darwin":
		return hasSuffix(filename, ".dylib") && contains(filename, expectedName)
	case "windows":
		return hasSuffix(filename, ".dll") && contains(filename, expectedName)
	default: // Linux
		return hasSuffix(filename, ".so") && contains(filename, expectedName)
	}
}

func (d *LibraryDownloader) detectVariant(filename string) string {
	if contains(filename, "avx512") {
		return "avx512"
	}
	if contains(filename, "avx2") {
		return "avx2"
	}
	if contains(filename, "avx") && !contains(filename, "avx2") && !contains(filename, "avx512") {
		return "avx"
	}
	return "fallback"
}

func (d *LibraryDownloader) selectBestVariant(candidates []LibraryAsset, platform *PlatformInfo) *LibraryAsset {
	// Always use fallback variant for maximum compatibility
	// This avoids SIGILL errors on CPUs that don't support AVX/AVX2/AVX512
	for _, c := range candidates {
		if c.Variant == "fallback" {
			return &c
		}
	}

	// Fallback to first available if no fallback found
	return &candidates[0]
}

// ProgressCallback is called during download to report progress
type ProgressCallback func(bytesComplete, totalBytes int64, mbps float64, done bool)

// Download downloads the library with resume support
func (d *LibraryDownloader) Download(asset *LibraryAsset) (string, error) {
	return d.DownloadWithProgress(asset, nil)
}

// DownloadWithProgress downloads the library with progress callback
func (d *LibraryDownloader) DownloadWithProgress(asset *LibraryAsset, progress ProgressCallback) (string, error) {
	// Ensure target directory exists
	if err := os.MkdirAll(d.targetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create target directory: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("GET", asset.URL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set output filename
	outputPath := filepath.Join(d.targetDir, asset.Name)

	// Create grab request
	req := &grab.Request{
		HTTPRequest: httpReq,
		Filename:    outputPath,
	}

	// Enable resume if file exists
	if info, err := os.Stat(outputPath); err == nil {
		// File exists, will auto-resume
		fmt.Printf("Resuming download from %d bytes\n", info.Size())
	}

	// Start download
	resp := d.client.Do(req)

	// Monitor progress if callback provided
	if progress != nil {
		startTime := time.Now()
		t := time.NewTicker(100 * time.Millisecond)
		defer t.Stop()

		for {
			select {
			case <-t.C:
				bytesComplete := resp.BytesComplete()
				totalBytes := resp.Size()
				elapsed := time.Since(startTime).Seconds()
				var mbps float64
				if elapsed > 0 {
					mbps = float64(bytesComplete) / (1024 * 1024) / elapsed
				}
				progress(bytesComplete, totalBytes, mbps, false)
			default:
				if resp.IsComplete() {
					bytesComplete := resp.BytesComplete()
					progress(bytesComplete, bytesComplete, 0, true)
					if err := resp.Err(); err != nil {
						return "", fmt.Errorf("download failed: %w", err)
					}
					return outputPath, nil
				}
				time.Sleep(50 * time.Millisecond)
			}
		}
	}

	// Wait for download to complete
	if err := resp.Err(); err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}

	return outputPath, nil
}

// DownloadLatest downloads the latest library for the current platform
func (d *LibraryDownloader) DownloadLatest() (string, error) {
	return d.DownloadLatestWithProgress(nil)
}

// DownloadLatestWithProgress downloads with progress callback
func (d *LibraryDownloader) DownloadLatestWithProgress(progress ProgressCallback) (string, error) {
	// Detect platform
	platform := DetectPlatform()
	fmt.Printf("Detected platform: %s/%s\n", platform.OS, platform.Arch)

	// Get latest release
	release, err := d.GetLatestRelease()
	if err != nil {
		return "", err
	}
	fmt.Printf("Latest release: %s\n", release.TagName)

	// Select best library
	asset, err := d.SelectBestLibrary(release, platform)
	if err != nil {
		return "", err
	}
	fmt.Printf("Selected library: %s (%s variant, %d bytes)\n",
		asset.Name, asset.Variant, asset.Size)

	// Download with progress
	path, err := d.DownloadWithProgress(asset, progress)
	if err != nil {
		return "", err
	}

	fmt.Printf("Library downloaded to: %s\n", path)
	return path, nil
}

// Helper functions
func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsAt(s, substr)
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
