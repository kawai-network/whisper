#ifndef GOWHISPER_H
#define GOWHISPER_H

#include <cstddef>
#include <cstdint>

// Windows DLL export/import macros
#ifdef _WIN32
  #ifdef GOWHISPER_EXPORTS
    #define GOWHISPER_API __declspec(dllexport)
  #else
    #define GOWHISPER_API __declspec(dllimport)
  #endif
#else
  #define GOWHISPER_API
#endif

extern "C" {
GOWHISPER_API int load_model(const char *const model_path);
GOWHISPER_API int load_model_vad(const char *const model_path);
GOWHISPER_API int vad(float pcmf32[], size_t pcmf32_size, float **segs_out,
        size_t *segs_out_len);
GOWHISPER_API int transcribe(uint32_t threads, char *lang, bool translate, bool tdrz,
               float pcmf32[], size_t pcmf32_len, size_t *segs_out_len,
               char *prompt);
GOWHISPER_API const char *get_segment_text(int i);
GOWHISPER_API int64_t get_segment_t0(int i);
GOWHISPER_API int64_t get_segment_t1(int i);
GOWHISPER_API int n_tokens(int i);
GOWHISPER_API int32_t get_token_id(int i, int j);
GOWHISPER_API bool get_segment_speaker_turn_next(int i);
}

#endif // GOWHISPER_H
