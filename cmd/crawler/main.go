// Command crawler is the crawler service. See docs/ARCHITECTURE.md.
package main

import (
	"fmt"
	"net/http"
	"os"

	"unipd-voice-assistant/internal/platform"
)

func main() {
	cfg := platform.LoadBase("crawler")
	mux := http.NewServeMux()

	// TODO(M2: scheduler, crawl frontier, parsers, chunker, change detection)

	if err := platform.Run(cfg, mux); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
