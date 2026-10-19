// Command rag is the rag service. See docs/ARCHITECTURE.md.
package main

import (
	"fmt"
	"net/http"
	"os"

	"unipd-voice-assistant/internal/platform"
)

func main() {
	cfg := platform.LoadBase("rag")
	mux := http.NewServeMux()

	// TODO(M3: POST /v1/ask (SSE), indexer worker, deadlines extraction)

	if err := platform.Run(cfg, mux); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
