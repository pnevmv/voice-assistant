# ADR-0009: Voice transport — SSE + base64 audio; batch STT first

**Status:** Accepted

## Context
Low time-to-first-audio is a priority, but this is a single-developer project that must be easy to test and evaluate.

## Decision
- Client → voice: `POST` of a recorded utterance (push-to-talk). Voice → client: one SSE stream with `transcript`, `citations`, `answer_text`, `audio` (base64 chunks), `done` events.
- **STT:** ElevenLabs `scribe_v3` (batch, one request per utterance) for M5. `scribe_v3_live` (realtime, WebSocket) is the later latency optimisation, done only if measured STT latency dominates time-to-first-audio. `scribe_v1` is deprecated and must not be used.
- **TTS:** primary `eleven_v4_turbo` (~100 ms median latency, 90+ languages), fallback `eleven_flash_v2_5`. One streaming request per sentence, with previous/next text for prosody continuity; sentences are released to TTS after the date guard. The fallback is used when the primary fails before the first audio byte. Both ids are environment variables (`ELEVENLABS_TTS_MODEL`, `ELEVENLABS_TTS_FALLBACK_MODEL`), so eval can run the same questions against either and compare time-to-first-audio.
- **Keyterms:** the STT parameter `keyterms` (supported by `scribe_v3`, `scribe_v3_live`, `scribe_v2`) biases recognition towards domain vocabulary: ISEE, ISEU, CFU, Uniweb, Unipd, bando, borsa di studio, immatricolazione, ESU, Ateneo, graduatoria, idoneo, fuori sede. The list lives in `config/keyterms.txt`, not in code.
  - Limits: batch up to 1000 terms; realtime up to 50 terms and 1000 characters in total; every term at most 50 characters and 5 words. For realtime the terms are passed as repeated `keyterms` query parameters on the WebSocket URL.
  - **The list is validated at voice-service startup against the realtime limits** (`speech.LoadKeyterms`; the service refuses to start otherwise). They are the strictest, so switching to `scribe_v3_live` needs no change to the list.
  - **Keyterms are billed extra** by ElevenLabs. The list is kept short, `ELEVENLABS_STT_KEYTERMS=false` disables them globally, the per-request `keyterms=false` override does so for one call (dev/eval), and eval measures whether they earn their cost.
  - **Per-request overrides (`keyterms`, `tts_model`) are off by default.** They are accepted only when `VOICE_ALLOW_OVERRIDES=true` (dev/eval); otherwise a request using one gets `400 overrides_disabled`.
- Citations travel as text on the same stream, never only as audio.

Alternative: WebSocket with binary frames (~33 % less bandwidth, full duplex, barge-in). Deferred; the internal pipeline (`transcribe → ask → sentence-split → synthesise`) is transport-agnostic.

## Consequences
- Curl/Python can drive and measure the whole flow; eval records TTFA at the client.
- Model ids and limits above were verified in the ElevenLabs docs on 2026-11-24. Providers change them, so they stay configuration, and `internal/speech` isolates the adapter; re-check them when M5 starts.
- Eval (M4/M5) compares **WER on the domain terms with vs without keyterms**, and time-to-first-audio for `eleven_v4_turbo` vs `eleven_flash_v2_5`, so the choice of model and the extra cost are decided by measurement.
