//go:build !windows
// +build !windows

package whisper

import (
	"github.com/ebitengine/purego"
)

// loadLibrary loads a shared library on Unix-like systems
func loadLibrary(path string) (uintptr, error) {
	return purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_GLOBAL)
}

// registerLibFunc is a wrapper around purego.RegisterLibFunc for Unix
func registerLibFunc(fn interface{}, lib uintptr, name string) {
	purego.RegisterLibFunc(fn, lib, name)
}

// closeLibrary unloads the shared library on Unix systems
func closeLibrary(handle uintptr) error {
	return purego.Dlclose(handle)
}
