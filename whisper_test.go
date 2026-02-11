package whisper

import (
	"os"
	"testing"
)

func TestLibraryLoading(t *testing.T) {
	// Test that we can load the library
	w, err := New(".")
	if err != nil {
		t.Fatalf("Failed to initialize whisper: %v", err)
	}
	if w == nil {
		t.Fatal("Expected non-nil Whisper instance")
	}
}

func TestModelLoading(t *testing.T) {
	modelPath := "test/data/ggml-tiny.en.bin"

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Skipf("Skipping test: model file not found at %s", modelPath)
	}

	w, err := New(".")
	if err != nil {
		t.Fatalf("Failed to initialize whisper: %v", err)
	}

	// Test loading transcription model
	if err := w.Load(modelPath); err != nil {
		t.Fatalf("Failed to load model: %v", err)
	}
}

func TestTranscribeBasic(t *testing.T) {
	modelPath := "test/data/ggml-tiny.en.bin"
	audioPath := "test/data/jfk.wav"

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Skipf("Skipping test: model file not found at %s", modelPath)
	}
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		t.Skipf("Skipping test: audio file not found at %s", audioPath)
	}

	w, err := New(".")
	if err != nil {
		t.Fatalf("Failed to initialize whisper: %v", err)
	}

	if err := w.Load(modelPath); err != nil {
		t.Fatalf("Failed to load model: %v", err)
	}

	opts := TranscriptionOptions{
		Language: "en",
		Threads:  1,
	}

	res, err := w.Transcribe(audioPath, opts)
	if err != nil {
		t.Fatalf("Failed to transcribe: %v", err)
	}

	// Assertions
	if len(res.Text) == 0 {
		t.Error("Expected transcription text, got empty string")
	}

	if len(res.Segments) == 0 {
		t.Error("Expected at least one segment")
	}

	// Log the result for manual verification
	t.Logf("Transcription: %s", res.Text)
	t.Logf("Number of segments: %d", len(res.Segments))
}

func TestTranscribeWithMultipleThreads(t *testing.T) {
	modelPath := "test/data/ggml-tiny.en.bin"
	audioPath := "test/data/jfk.wav"

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Skipf("Skipping test: model file not found at %s", modelPath)
	}
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		t.Skipf("Skipping test: audio file not found at %s", audioPath)
	}

	w, err := New(".")
	if err != nil {
		t.Fatalf("Failed to initialize whisper: %v", err)
	}

	if err := w.Load(modelPath); err != nil {
		t.Fatalf("Failed to load model: %v", err)
	}

	opts := TranscriptionOptions{
		Language: "en",
		Threads:  4, // Use multiple threads
	}

	res, err := w.Transcribe(audioPath, opts)
	if err != nil {
		t.Fatalf("Failed to transcribe with multiple threads: %v", err)
	}

	if len(res.Text) == 0 {
		t.Error("Expected transcription text, got empty string")
	}

	t.Logf("Transcription (4 threads): %s", res.Text)
}

func TestTranscribeWithDiarization(t *testing.T) {
	modelPath := "test/data/ggml-tiny.en.bin"
	audioPath := "test/data/jfk.wav"

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Skipf("Skipping test: model file not found at %s", modelPath)
	}
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		t.Skipf("Skipping test: audio file not found at %s", audioPath)
	}

	w, err := New(".")
	if err != nil {
		t.Fatalf("Failed to initialize whisper: %v", err)
	}

	if err := w.Load(modelPath); err != nil {
		t.Fatalf("Failed to load model: %v", err)
	}

	opts := TranscriptionOptions{
		Language: "en",
		Threads:  2,
		Diarize:  true, // Enable speaker diarization
	}

	res, err := w.Transcribe(audioPath, opts)
	if err != nil {
		t.Fatalf("Failed to transcribe with diarization: %v", err)
	}

	if len(res.Text) == 0 {
		t.Error("Expected transcription text, got empty string")
	}

	t.Logf("Transcription (with diarization): %s", res.Text)
}

func TestSegmentDetails(t *testing.T) {
	modelPath := "test/data/ggml-tiny.en.bin"
	audioPath := "test/data/jfk.wav"

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Skipf("Skipping test: model file not found at %s", modelPath)
	}
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		t.Skipf("Skipping test: audio file not found at %s", audioPath)
	}

	w, err := New(".")
	if err != nil {
		t.Fatalf("Failed to initialize whisper: %v", err)
	}

	if err := w.Load(modelPath); err != nil {
		t.Fatalf("Failed to load model: %v", err)
	}

	opts := TranscriptionOptions{
		Language: "en",
		Threads:  1,
	}

	res, err := w.Transcribe(audioPath, opts)
	if err != nil {
		t.Fatalf("Failed to transcribe: %v", err)
	}

	// Check segment details
	for i, seg := range res.Segments {
		if seg.Start >= seg.End {
			t.Errorf("Segment %d: start time (%d) should be less than end time (%d)", i, seg.Start, seg.End)
		}
		if len(seg.Text) == 0 {
			t.Errorf("Segment %d: expected non-empty text", i)
		}
		if seg.Id != int32(i) {
			t.Errorf("Segment %d: expected ID %d, got %d", i, i, seg.Id)
		}

		t.Logf("Segment %d: [%d-%d] %s (tokens: %d)", i, seg.Start, seg.End, seg.Text, len(seg.Tokens))
	}
}
