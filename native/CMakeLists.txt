cmake_minimum_required(VERSION 3.12)
project(gowhisper LANGUAGES C CXX)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

add_subdirectory(../sources/whisper.cpp whisper-build)

# Define export macro for Windows DLL
if(WIN32)
    add_definitions(-DGOWHISPER_EXPORTS)
endif()

add_library(gowhisper SHARED gowhisper.cpp)
target_link_libraries(gowhisper PRIVATE whisper ggml)

if(CMAKE_CXX_COMPILER_ID MATCHES "GNU" AND CMAKE_CXX_COMPILER_VERSION VERSION_LESS 9.0)
    target_link_libraries(gosd PRIVATE stdc++fs)
endif()

set_property(TARGET gowhisper PROPERTY CXX_STANDARD 17)
set_target_properties(gowhisper PROPERTIES LIBRARY_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR})

# Windows-specific settings
if(WIN32)
    set_target_properties(gowhisper PROPERTIES
        RUNTIME_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}
        ARCHIVE_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}
    )
    # On Windows, the output is a .dll file
    if(MSVC)
        target_compile_options(gowhisper PRIVATE /W4)
        # Use static runtime to avoid dependency on VC++ Redistributable
        set(CMAKE_MSVC_RUNTIME_LIBRARY "MultiThreaded$<$<CONFIG:Debug>:Debug>")
        target_compile_options(gowhisper PRIVATE /MT)
    endif()
endif()
