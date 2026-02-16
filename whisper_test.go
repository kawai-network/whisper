package whisper

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func skipIfNoLibrary(t *testing.T) {
	t.Helper()
	libName := LibraryName(runtime.GOOS)
	libFile := filepath.Join(".", libName)
	if _, err := os.Stat(libFile); os.IsNotExist(err) {
		t.Skipf("Skipping test: library not found at %s. Download from https://github.com/kawai-network/whisper/releases/latest", libFile)
	}
}

func skipIfNoModel(t *testing.T, modelPath string) {
	t.Helper()
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Skipf("Skipping test: model file not found at %s", modelPath)
	}
}

func skipIfNoAudio(t *testing.T, audioPath string) {
	t.Helper()
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		t.Skipf("Skipping test: audio file not found at %s", audioPath)
	}
}

func TestLibraryLoading(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}
	skipIfNoLibrary(t)

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
	skipIfNoLibrary(t)
	skipIfNoModel(t, modelPath)

	w, err := New(".")
	if err != nil {
		t.Fatalf("Failed to initialize whisper: %v", err)
	}

	if err := w.Load(modelPath); err != nil {
		t.Fatalf("Failed to load model: %v", err)
	}
}

func TestTranscribeBasic(t *testing.T) {
	modelPath := "test/data/ggml-tiny.en.bin"
	audioPath := "test/data/jfk.wav"
	skipIfNoLibrary(t)
	skipIfNoModel(t, modelPath)
	skipIfNoAudio(t, audioPath)

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

	if len(res.Text) == 0 {
		t.Error("Expected transcription text, got empty string")
	}

	if len(res.Segments) == 0 {
		t.Error("Expected at least one segment")
	}

	t.Logf("Transcription: %s", res.Text)
	t.Logf("Number of segments: %d", len(res.Segments))
}

func TestTranscribeWithMultipleThreads(t *testing.T) {
	modelPath := "test/data/ggml-tiny.en.bin"
	audioPath := "test/data/jfk.wav"
	skipIfNoLibrary(t)
	skipIfNoModel(t, modelPath)
	skipIfNoAudio(t, audioPath)

	w, err := New(".")
	if err != nil {
		t.Fatalf("Failed to initialize whisper: %v", err)
	}

	if err := w.Load(modelPath); err != nil {
		t.Fatalf("Failed to load model: %v", err)
	}

	opts := TranscriptionOptions{
		Language: "en",
		Threads:  4,
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
	skipIfNoLibrary(t)
	skipIfNoModel(t, modelPath)
	skipIfNoAudio(t, audioPath)

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
		Diarize:  true,
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
	skipIfNoLibrary(t)
	skipIfNoModel(t, modelPath)
	skipIfNoAudio(t, audioPath)

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

func TestInvalidModelPath(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}
	skipIfNoLibrary(t)

	w, err := New(".")
	if err != nil {
		t.Fatalf("Failed to initialize whisper: %v", err)
	}

	err = w.Load("non_existent_model.bin")
	if err == nil {
		t.Error("Expected error when loading non-existent model, got nil")
	}
}

func TestInvalidAudioPath(t *testing.T) {
	modelPath := "test/data/ggml-tiny.en.bin"
	skipIfNoLibrary(t)
	skipIfNoModel(t, modelPath)

	w, err := New(".")
	if err != nil {
		t.Fatalf("Failed to initialize whisper: %v", err)
	}

	if err := w.Load(modelPath); err != nil {
		t.Fatalf("Failed to load model: %v", err)
	}

	_, err = w.Transcribe("non_existent_audio.wav", TranscriptionOptions{})
	if err == nil {
		t.Error("Expected error when transcribing non-existent audio, got nil")
	}
}
