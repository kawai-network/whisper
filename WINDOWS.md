# Windows DLL Support

This document explains how to build the Whisper library for Windows as a DLL.

## Prerequisites

### Option 1: Cross-compilation from Linux/macOS (Recommended for CI/CD)

1. Install MinGW-w64 cross-compiler:
   - **Ubuntu/Debian**: `sudo apt-get install mingw-w64`
   - **macOS**: `brew install mingw-w64`
   - **Fedora**: `sudo dnf install mingw64-gcc mingw64-gcc-c++`

2. CMake (version 3.12 or higher)

### Option 2: Build natively on Windows

1. Visual Studio 2022 with C++ workload
2. CMake (bundled with VS or standalone)
3. Git for Windows

## Building

### Method 1: Cross-compilation from Linux/macOS

```bash
# Build Windows DLL using MinGW
make gowhisper-fallback.dll

# Or specify custom compiler paths
make gowhisper-fallback.dll WINDOWS_CC=x86_64-w64-mingw32-gcc WINDOWS_CXX=x86_64-w64-mingw32-g++
```

This will produce `gowhisper-fallback.dll` that can be used on Windows.

### Method 2: Native Windows build with MSVC

Open Developer Command Prompt for Visual Studio 2022:

```cmd
git clone <repository-url>
cd whisper
make sources/whisper.cpp
make gowhisper-fallback-msvc.dll
```

This will produce `gowhisper-fallback-msvc.dll`.

### Method 3: Manual CMake build on Windows

```cmd
git clone <repository-url>
cd whisper
make sources/whisper.cpp
mkdir build-windows
cd build-windows
cmake ../native -G "Visual Studio 17 2022" -A x64 -DGGML_AVX=OFF -DGGML_AVX2=OFF -DGGML_AVX512=OFF
cmake --build . --config Release
copy Release\gowhisper.dll ..\gowhisper.dll
```

## Using the DLL

Place the `gowhisper-fallback.dll` (or `gowhisper.dll`) in your application directory or in a directory specified when creating the Whisper instance:

```go
import "github.com/kawai-network/whisper"

// The library will automatically find gowhisper-fallback.dll on Windows
w, err := whisper.New(".") // Look in current directory
if err != nil {
    log.Fatal(err)
}
```

## Architecture Support

Currently, Windows builds only support the **fallback** (generic) variant. CPU-specific optimizations (AVX, AVX2, AVX512) are supported on Linux only.

## Distribution

The Windows DLL can be distributed with your application. Required files:

- `gowhisper-fallback.dll` - The main library
- Model files (e.g., `ggml-base.bin`) - Required for transcription

## Notes

- The DLL is built with the `GOWHISPER_EXPORTS` macro defined, which exports all necessary functions
- On Windows, Go's `purego` library will load the `.dll` file automatically
- The DLL has no external dependencies beyond standard Windows libraries
- File extensions:
  - Windows: `.dll` (e.g., `gowhisper-fallback.dll`)
  - Linux: `.so` (e.g., `libgowhisper-fallback.so`)
  - macOS: `.dylib` (e.g., `libgowhisper-fallback.dylib`)
