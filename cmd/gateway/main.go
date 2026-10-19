// Command gateway is the gateway service. See docs/ARCHITECTURE.md.
package main

import (
	"fmt"
	"net/http"
	"os"

	"unipd-voice-assistant/internal/platform"
)

func main() {
	cfg := platform.LoadBase("gateway")
	mux := http.NewServeMux()

	// TODO(M1: /v1/chat/completions, /v1/embeddings, provider chain, cache, metrics)

	if err := platform.Run(cfg, mux); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
