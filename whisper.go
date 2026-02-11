package whisper

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"github.com/go-audio/wav"
)

// Whisper struct encapsulates the library instance and its methods
type Whisper struct {
	// Function pointers to be loaded from the shared library
	cppLoadModel                 func(modelPath string) int
	cppLoadModelVAD              func(modelPath string) int
	cppVAD                       func(pcmf32 []float32, pcmf32Size uintptr, segsOut unsafe.Pointer, segsOutLen unsafe.Pointer) int
	cppTranscribe                func(threads uint32, lang string, translate bool, diarize bool, pcmf32 []float32, pcmf32Len uintptr, segsOutLen unsafe.Pointer, prompt string) int
	cppGetSegmentText            func(i int) string
	cppGetSegmentStart           func(i int) int64
	cppGetSegmentEnd             func(i int) int64
	cppNTokens                   func(i int) int
	cppGetTokenID                func(i int, j int) int
	cppGetSegmentSpeakerTurnNext func(i int) bool
	libHandle                    uintptr
}

// New creates a new Whisper instance.
// If libPath is a file, it loads that file.
// If libPath is a directory, it attempts to find the best available library in that directory.
// If libPath is empty, it attempts to find the best available library in the current directory or defaults.
// If no library is found, it automatically downloads the latest release.
func New(libPath string) (*Whisper, error) {
	w := &Whisper{}

	var path string
	var err error

	if libPath == "" {
		libPath = "."
	}

	info, err := os.Stat(libPath)
	if err == nil && info.IsDir() {
		path = findBestLibrary(libPath)
		if path == "" {
			// Library not found, try to auto-download
			fmt.Printf("Library not found in %s, attempting to download...\n", libPath)
			downloader := NewLibraryDownloader(libPath)
			path, err = downloader.DownloadLatest()
			if err != nil {
				return nil, fmt.Errorf("no suitable whisper library found in %s and auto-download failed: %w", libPath, err)
			}
			fmt.Printf("Library downloaded to: %s\n", path)
		}
	} else {
		path = libPath
	}

	// Load the library
	// Convert to absolute path to ensure dlopen can find it
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for %s: %w", path, err)
	}

	lib, err := loadLibrary(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open library at %s: %w", absPath, err)
	}

	// Register function pointers
	registerLibFunc(&w.cppLoadModel, lib, "load_model")
	registerLibFunc(&w.cppLoadModelVAD, lib, "load_model_vad")
	registerLibFunc(&w.cppVAD, lib, "vad")
	registerLibFunc(&w.cppTranscribe, lib, "transcribe")
	registerLibFunc(&w.cppGetSegmentText, lib, "get_segment_text")
	registerLibFunc(&w.cppGetSegmentStart, lib, "get_segment_t0")
	registerLibFunc(&w.cppGetSegmentEnd, lib, "get_segment_t1")
	registerLibFunc(&w.cppNTokens, lib, "n_tokens")
	registerLibFunc(&w.cppGetTokenID, lib, "get_token_id")
	registerLibFunc(&w.cppGetSegmentSpeakerTurnNext, lib, "get_segment_speaker_turn_next")

	w.libHandle = lib

	return w, nil
}

// Close closes the Whisper instance and unloads the library
func (w *Whisper) Close() error {
	if w.libHandle != 0 {
		return closeLibrary(w.libHandle)
	}
	return nil
}

func findBestLibrary(dir string) string {
	// Platform-specific library extensions
	ext := ".so"
	prefix := "lib"

	switch runtime.GOOS {
	case "darwin":
		// macOS uses .dylib or .so
		ext = ".dylib"
	case "windows":
		// Windows uses .dll
		ext = ".dll"
		prefix = ""
	}

	// Always use fallback variant for maximum compatibility
	// This avoids SIGILL errors on CPUs that don't support AVX/AVX2/AVX512
	path := filepath.Join(dir, prefix+"gowhisper-fallback"+ext)
	if _, err := os.Stat(path); err == nil {
		return path
	}

	return ""
}

// ModelOptions represents options for loading a model
type ModelOptions struct {
	ModelFile string
	Type      string // "vad" or "transcription" (inferred or explicit)
}

// Load loads the model from the specified file
func (w *Whisper) Load(modelPath string) error {
	// This simplifies the logic. Original code accepted generic "options" but only checked for "vad_only".
	// We can assume based on usage or just try loading.
	// For now, let's just expose LoadModel and LoadModelVAD separately or via a flag.
	if ret := w.cppLoadModel(modelPath); ret != 0 {
		return fmt.Errorf("failed to load Whisper transcription model from %s", modelPath)
	}
	return nil
}

// LoadVAD loads the VAD model
func (w *Whisper) LoadVAD(modelPath string) error {
	if ret := w.cppLoadModelVAD(modelPath); ret != 0 {
		return fmt.Errorf("failed to load Whisper VAD model from %s", modelPath)
	}
	return nil
}

// VADSegment represents a voice activity detection segment
type VADSegment struct {
	Start float32
	End   float32
}

// VAD performs voice activity detection
func (w *Whisper) VAD(audio []float32) ([]VADSegment, error) {
	// We expect 0xdeadbeef to be overwritten and if we see it in a stack trace we know it wasn't
	segsPtr, segsLen := uintptr(0xdeadbeef), uintptr(0xdeadbeef)
	segsPtrPtr, segsLenPtr := unsafe.Pointer(&segsPtr), unsafe.Pointer(&segsLen)

	if ret := w.cppVAD(audio, uintptr(len(audio)), segsPtrPtr, segsLenPtr); ret != 0 {
		return nil, fmt.Errorf("failed VAD execution")
	}

	// Happens when CPP vector has not had any elements pushed to it
	if segsPtr == 0 {
		return []VADSegment{}, nil
	}

	// unsafeptr warning is caused by segsPtr being on the stack and therefor being subject to stack copying AFAICT
	// however the stack shouldn't have grown between setting segsPtr and now, also the memory pointed to is allocated by C++
	segs := unsafe.Slice((*float32)(unsafe.Pointer(segsPtr)), segsLen)

	vadSegments := []VADSegment{}
	for i := range len(segs) >> 1 {
		s := segs[2*i] / 100
		t := segs[2*i+1] / 100
		vadSegments = append(vadSegments, VADSegment{
			Start: s,
			End:   t,
		})
	}

	return vadSegments, nil
}

// TranscriptionOptions configuration for transcription
type TranscriptionOptions struct {
	Threads   uint32
	Language  string
	Translate bool
	Diarize   bool
	Prompt    string
}

// Segment represents a transcribed segment
type Segment struct {
	Id     int32
	Text   string
	Start  int64
	End    int64
	Tokens []int32
}

// TranscriptionResult result of transcription
type TranscriptionResult struct {
	Segments []*Segment
	Text     string
}

// Transcribe transcribes the audio file
func (w *Whisper) Transcribe(audioFile string, opts TranscriptionOptions) (TranscriptionResult, error) {
	// Convert audio to appropriate format (16kHz wav)
	// We use a temp file for conversion
	dir, err := os.MkdirTemp("", "whisper")
	if err != nil {
		return TranscriptionResult{}, err
	}
	defer os.RemoveAll(dir)

	convertedPath := filepath.Join(dir, "converted.wav")

	// Use internal helper to convert audio
	if err := audioToWav(audioFile, convertedPath); err != nil {
		return TranscriptionResult{}, fmt.Errorf("failed to convert audio: %w", err)
	}

	// Open samples
	fh, err := os.Open(convertedPath)
	if err != nil {
		return TranscriptionResult{}, err
	}
	defer fh.Close()

	// Read samples
	d := wav.NewDecoder(fh)
	buf, err := d.FullPCMBuffer()
	if err != nil {
		return TranscriptionResult{}, err
	}

	data := buf.AsFloat32Buffer().Data
	segsLen := uintptr(0xdeadbeef)
	segsLenPtr := unsafe.Pointer(&segsLen)

	if ret := w.cppTranscribe(opts.Threads, opts.Language, opts.Translate, opts.Diarize, data, uintptr(len(data)), segsLenPtr, opts.Prompt); ret != 0 {
		return TranscriptionResult{}, fmt.Errorf("failed Transcribe execution")
	}

	segments := []*Segment{}
	text := ""
	for i := range int(segsLen) {
		// segment start/end conversion factor taken from https://github.com/ggml-org/whisper.cpp/blob/master/examples/cli/cli.cpp#L895
		s := w.cppGetSegmentStart(i) * (10000000)
		t := w.cppGetSegmentEnd(i) * (10000000)

		// Copy string to avoid memory issues if C++ frees it (though purego usually copies)
		txt := w.cppGetSegmentText(i)
		// txt := strings.Clone(w.cppGetSegmentText(i)) // Clone if needed, but purego string marshaling typically creates a go string copy?
		// Actually, purego converts *char to string by copying.

		tokens := make([]int32, w.cppNTokens(i))

		if opts.Diarize && w.cppGetSegmentSpeakerTurnNext(i) {
			txt += " [SPEAKER_TURN]"
		}

		for j := range tokens {
			tokens[j] = int32(w.cppGetTokenID(i, j))
		}
		segment := &Segment{
			Id:    int32(i),
			Text:  txt,
			Start: s, End: t,
			Tokens: tokens,
		}

		segments = append(segments, segment)

		text += " " + strings.TrimSpace(txt)
	}

	return TranscriptionResult{
		Segments: segments,
		Text:     strings.TrimSpace(text),
	}, nil
}

// audioToWav converts input audio to 16kHz WAV using ffmpeg
func audioToWav(src, dst string) error {
	cmd := exec.Command("ffmpeg", "-y", "-i", src, "-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le", dst)
	// Check if ffmpeg is seemingly available or just run it.
	// If user doesn't have ffmpeg, this will fail.
	// We could check error output.
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed: %s: %s", err, string(output))
	}
	return nil
}
