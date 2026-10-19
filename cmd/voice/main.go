// Command voice is the voice service. See docs/ARCHITECTURE.md.
package main

import (
	"fmt"
	"net/http"
	"os"

	"unipd-voice-assistant/internal/platform"
	"unipd-voice-assistant/internal/speech"
)

func main() {
	cfg := platform.LoadBase("voice")

	if err := speech.ValidateSTTModel(platform.String("ELEVENLABS_STT_MODEL", "scribe_v3")); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Fail fast on a bad keyterms list: it is validated against the REALTIME
	// limits (scribe_v3_live), so a later switch of STT model needs no change.
	keyterms, err := speech.LoadKeyterms(platform.String("ELEVENLABS_KEYTERMS_FILE", "config/keyterms.txt"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	log := platform.NewLogger(cfg)
	log.Info("keyterms loaded", "count", len(keyterms))

	// Per-request overrides (keyterms, tts_model) exist for eval/dev only. They
	// are OFF unless explicitly enabled; when off, a request that sets one is
	// rejected with 400 overrides_disabled (never silently ignored, so an eval
	// can't measure the wrong configuration without noticing).
	allowOverrides := platform.Bool("VOICE_ALLOW_OVERRIDES", false)
	if allowOverrides {
		log.Warn("VOICE_ALLOW_OVERRIDES=true: per-request keyterms/tts_model overrides enabled (dev/eval only)")
	}
	_ = allowOverrides // used by the M5 handler

	mux := http.NewServeMux()

	// TODO(M5: POST /v1/voice/ask (SSE): STT -> RAG -> streaming TTS)

	if err := platform.Run(cfg, mux); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
