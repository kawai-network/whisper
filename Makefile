CMAKE_ARGS?=
BUILD_TYPE?=
NATIVE?=false

GOCMD?=go
GO_TAGS?=
JOBS?=$(shell nproc --ignore=1 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)

# whisper.cpp version
WHISPER_REPO?=https://github.com/ggml-org/whisper.cpp
WHISPER_CPP_VERSION?=764482c3175d9c3bc6089c1ec84df7d1b9537d83
SO_TARGET?=libgowhisper.so

CMAKE_ARGS+=-DBUILD_SHARED_LIBS=OFF

ifeq ($(NATIVE),false)
	CMAKE_ARGS+=-DGGML_NATIVE=OFF
endif

ifeq ($(BUILD_TYPE),cublas)
	CMAKE_ARGS+=-DGGML_CUDA=ON
else ifeq ($(BUILD_TYPE),openblas)
	CMAKE_ARGS+=-DGGML_BLAS=ON -DGGML_BLAS_VENDOR=OpenBLAS
else ifeq ($(BUILD_TYPE),clblas)
	CMAKE_ARGS+=-DGGML_CLBLAST=ON -DCLBlast_DIR=/some/path
else ifeq ($(BUILD_TYPE),hipblas)
	CMAKE_ARGS+=-DGGML_HIPBLAS=ON
else ifeq ($(BUILD_TYPE),vulkan)
	CMAKE_ARGS+=-DGGML_VULKAN=ON
else ifeq ($(OS),Darwin)
	ifneq ($(BUILD_TYPE),metal)
		CMAKE_ARGS+=-DGGML_METAL=OFF
	else
		CMAKE_ARGS+=-DGGML_METAL=ON
		CMAKE_ARGS+=-DGGML_METAL_EMBED_LIBRARY=ON
	endif
endif

ifeq ($(BUILD_TYPE),sycl_f16)
	CMAKE_ARGS+=-DGGML_SYCL=ON \
		-DCMAKE_C_COMPILER=icx \
		-DCMAKE_CXX_COMPILER=icpx \
		-DGGML_SYCL_F16=ON
endif

ifeq ($(BUILD_TYPE),sycl_f32)
	CMAKE_ARGS+=-DGGML_SYCL=ON \
		-DCMAKE_C_COMPILER=icx \
		-DCMAKE_CXX_COMPILER=icpx
endif

sources/whisper.cpp:
	mkdir -p sources/whisper.cpp
	cd sources/whisper.cpp && \
	git init && \
	git remote add origin $(WHISPER_REPO) && \
	git fetch origin && \
	git checkout $(WHISPER_CPP_VERSION) && \
	git submodule update --init --recursive --depth 1 --single-branch

# Detect OS
UNAME_S := $(shell uname -s)

# Platform detection for cross-compilation
ifeq ($(TARGET_OS),windows)
	PLATFORM_EXT = .dll
	PLATFORM_PREFIX = 
else ifeq ($(UNAME_S),Darwin)
	PLATFORM_EXT = .dylib
	PLATFORM_PREFIX = lib
else
	PLATFORM_EXT = .so
	PLATFORM_PREFIX = lib
endif

# Only build CPU variants on Linux
ifeq ($(UNAME_S),Linux)
	VARIANT_TARGETS = libgowhisper-avx.so libgowhisper-avx2.so libgowhisper-avx512.so libgowhisper-fallback.so
else ifeq ($(UNAME_S),Darwin)
	# macOS uses dylib
	VARIANT_TARGETS = libgowhisper-fallback.dylib
else
	VARIANT_TARGETS =
endif

# Windows cross-compile targets
WINDOWS_TARGETS = gowhisper-fallback.dll

whisper: whisper.go $(VARIANT_TARGETS)
	CGO_ENABLED=0 $(GOCMD) build -tags "$(GO_TAGS)" ./...

package: whisper
	bash package.sh

build: package

clean: purge
	rm -rf libgowhisper*.so libgowhisper*.dylib gowhisper*.dll package sources/whisper.cpp whisper

purge:
	rm -rf build*

# Build all variants (Linux only)
ifeq ($(UNAME_S),Linux)
libgowhisper-avx.so: sources/whisper.cpp
	$(MAKE) purge
	$(info ${GREEN}I whisper build info:avx${RESET})
	SO_TARGET=libgowhisper-avx.so CMAKE_ARGS="$(CMAKE_ARGS) -DGGML_AVX=on -DGGML_AVX2=off -DGGML_AVX512=off -DGGML_FMA=off -DGGML_F16C=off -DGGML_BMI2=off" $(MAKE) libgowhisper-custom
	rm -rfv build*

libgowhisper-avx2.so: sources/whisper.cpp
	$(MAKE) purge
	$(info ${GREEN}I whisper build info:avx2${RESET})
	SO_TARGET=libgowhisper-avx2.so CMAKE_ARGS="$(CMAKE_ARGS) -DGGML_AVX=on -DGGML_AVX2=on -DGGML_AVX512=off -DGGML_FMA=on -DGGML_F16C=on -DGGML_BMI2=on" $(MAKE) libgowhisper-custom
	rm -rfv build*

libgowhisper-avx512.so: sources/whisper.cpp
	$(MAKE) purge
	$(info ${GREEN}I whisper build info:avx512${RESET})
	SO_TARGET=libgowhisper-avx512.so CMAKE_ARGS="$(CMAKE_ARGS) -DGGML_AVX=on -DGGML_AVX2=on -DGGML_AVX512=on -DGGML_FMA=on -DGGML_F16C=on -DGGML_BMI2=on" $(MAKE) libgowhisper-custom
	rm -rfv build*
endif

# Build fallback variant (Linux)
libgowhisper-fallback.so: sources/whisper.cpp
	$(MAKE) purge
	$(info ${GREEN}I whisper build info:fallback${RESET})
	SO_TARGET=libgowhisper-fallback.so CMAKE_ARGS="$(CMAKE_ARGS) -DGGML_AVX=off -DGGML_AVX2=off -DGGML_AVX512=off -DGGML_FMA=off -DGGML_F16C=off -DGGML_BMI2=off" $(MAKE) libgowhisper-custom
	rm -rfv build*

# Build fallback variant (macOS)
libgowhisper-fallback.dylib: sources/whisper.cpp
	$(MAKE) purge
	$(info ${GREEN}I whisper build info:fallback (macOS)${RESET})
	SO_TARGET=libgowhisper-fallback.dylib CMAKE_ARGS="$(CMAKE_ARGS) -DGGML_METAL=OFF -DGGML_AVX=off -DGGML_AVX2=off -DGGML_AVX512=off -DGGML_FMA=off -DGGML_F16C=off -DGGML_BMI2=off" $(MAKE) libgowhisper-custom
	rm -rfv build*

# Windows cross-compilation using MinGW
# Usage: make gowhisper-fallback.dll WINDOWS_CC=x86_64-w64-mingw32-gcc WINDOWS_CXX=x86_64-w64-mingw32-g++
WINDOWS_CC?=x86_64-w64-mingw32-gcc
WINDOWS_CXX?=x86_64-w64-mingw32-g++
WINDOWS_STRIP?=x86_64-w64-mingw32-strip

gowhisper-fallback.dll: sources/whisper.cpp
	$(MAKE) purge
	$(info ${GREEN}I whisper build info: Windows DLL (fallback)${RESET})
	mkdir -p build-windows && \
	cd build-windows && \
	cmake ../native \
		-DCMAKE_TOOLCHAIN_FILE=../cmake/WindowsToolchain.cmake \
		-DCMAKE_C_COMPILER=$(WINDOWS_CC) \
		-DCMAKE_CXX_COMPILER=$(WINDOWS_CXX) \
		$(CMAKE_ARGS) \
		-DGGML_AVX=off \
		-DGGML_AVX2=off \
		-DGGML_AVX512=off \
		-DGGML_FMA=off \
		-DGGML_F16C=off \
		-DGGML_BMI2=off && \
	cmake --build . --config Release -j$(JOBS) && \
	cd .. && \
	cp build-windows/gowhisper.dll ./gowhisper-fallback.dll && \
	$(WINDOWS_STRIP) ./gowhisper-fallback.dll || true
	rm -rfv build-windows

# Windows build with MSVC (requires Visual Studio environment)
gowhisper-fallback-msvc.dll: sources/whisper.cpp
	$(MAKE) purge
	$(info ${GREEN}I whisper build info: Windows DLL MSVC (fallback)${RESET})
	mkdir -p build-windows-msvc && \
	cd build-windows-msvc && \
	cmake ../native \
		-G "Visual Studio 17 2022" \
		-A x64 \
		$(CMAKE_ARGS) \
		-DGGML_AVX=off \
		-DGGML_AVX2=off \
		-DGGML_AVX512=off \
		-DGGML_FMA=off \
		-DGGML_F16C=off \
		-DGGML_BMI2=off && \
	cmake --build . --config Release && \
	cd .. && \
	cp build-windows-msvc/Release/gowhisper.dll ./gowhisper-fallback-msvc.dll
	rm -rfv build-windows-msvc

windows: $(WINDOWS_TARGETS)

libgowhisper-custom: native/CMakeLists.txt native/gowhisper.cpp native/gowhisper.h
	mkdir -p build-$(SO_TARGET) && \
	cd build-$(SO_TARGET) && \
	cmake ../native $(CMAKE_ARGS) && \
	cmake --build . --config Release -j$(JOBS) && \
	cd .. && \
	if [ -f build-$(SO_TARGET)/libgowhisper.so ]; then mv build-$(SO_TARGET)/libgowhisper.so ./$(SO_TARGET); fi && \
	if [ -f build-$(SO_TARGET)/libgowhisper.dylib ]; then mv build-$(SO_TARGET)/libgowhisper.dylib ./$(SO_TARGET); fi && \
	if [ -f build-$(SO_TARGET)/gowhisper.dll ]; then mv build-$(SO_TARGET)/gowhisper.dll ./$(SO_TARGET); fi

all: whisper package

.PHONY: whisper package build clean purge windows libgowhisper-custom
