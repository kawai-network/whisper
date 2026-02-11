//go:build windows
// +build windows

package whisper

import (
	"fmt"
	"runtime"

	"github.com/ebitengine/purego"
	"golang.org/x/sys/windows"
)

// loadLibrary loads a DLL on Windows
func loadLibrary(path string) (uintptr, error) {
	handle, err := windows.LoadLibrary(path)
	if err != nil {
		return 0, fmt.Errorf("failed to load library: %w", err)
	}
	return uintptr(handle), nil
}

// registerLibFunc is a wrapper around purego.RegisterLibFunc that works with Windows handles
func registerLibFunc(fn interface{}, lib uintptr, name string) {
	// On Windows, we need to get the procedure address first
	handle := windows.Handle(lib)
	proc, err := windows.GetProcAddress(handle, name)
	if err != nil {
		return
	}
	// Use purego.RegisterFunc for Windows
	purego.RegisterFunc(fn, proc)
}

// init registers the Go runtime for purego
func init() {
	// Ensure we're on Windows
	if runtime.GOOS != "windows" {
		panic("this file should only be compiled on Windows")
	}
}
