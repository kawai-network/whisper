package whisper

import (
	"fmt"
	"os"
	"testing"
)

func TestTranscribe(t *testing.T) {
	// 1. Setup paths
	// We need a model file and an audio file.
	// For CI/testing, we can download tiny model and a short audio sample.
	modelPath := "test/data/ggml-tiny.en.bin"
	audioPath := "test/data/jfk.wav"

	// Ensure test data exists (in a real scenario, you'd likely have a setup step or standard test data)
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Skipf("Skipping test: model file not found at %s. Please download a model (e.g. ggml-tiny.en.bin) to run this test.", modelPath)
	}
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		t.Skipf("Skipping test: audio file not found at %s. Please provide a waiver audio file.", audioPath)
	}

	// 2. Initialize Whisper
	// We assume libraries are built in the current directory or a known location.
	// In CI, we will ensure they are built.
	libPath := "." // Start looking in current dir
	w, err := New(libPath)
	if err != nil {
		t.Fatalf("Failed to initialize whisper: %v", err)
	}

	// 3. Load Model
	if err := w.Load(modelPath); err != nil {
		t.Fatalf("Failed to load model: %v", err)
	}

	// 4. Run Transcription
	// Use minimal options for testing
	opts := TranscriptionOptions{
		Language: "en",
		Threads:  1,
	}

	res, err := w.Transcribe(audioPath, opts)
	if err != nil {
		t.Fatalf("Failed to transcribe: %v", err)
	}

	// 5. Assertions
	fmt.Printf("Transcription: %s\n", res.Text)
	if len(res.Text) == 0 {
		t.Error("Expected transcription text, got empty string")
	}

	// Basic check for content (JFK sample usually contains "Ask not what your country...")
	// We do a loose check.
	if len(res.Segments) == 0 {
		t.Error("Expected at least one segment")
	}
}
