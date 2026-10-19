package ingest

import (
	"errors"
	"testing"

	"github.com/temoto/robotstxt"
)

const unipdRobots = "User-Agent: *\nAllow: /\n\nSitemap: https://www.unipd.it/sitemap.xml\n"

func TestRobotsFromResponse(t *testing.T) {
	const ua = "UniPDAssistantBot"
	tests := []struct {
		name      string
		status    int
		body      string
		fetchErr  error
		wantErr   bool // ErrRobotsUnavailable
		path      string
		wantAllow bool
	}{
		{"www.unipd.it style: allow all", 200, unipdRobots, nil, false, "/borse-studio-alloggi", true},
		{"200 with a disallow rule", 200, "User-agent: *\nDisallow: /private\n", nil, false, "/private/x", false},
		{"200 rule does not hit other paths", 200, "User-agent: *\nDisallow: /private\n", nil, false, "/public", true},
		{"200 rule aimed at our bot", 200, "User-agent: UniPDAssistantBot\nDisallow: /\n", nil, false, "/", false},
		{"200 empty file allows everything", 200, "", nil, false, "/x", true},
		// 4xx: no restrictions (RFC 9309 §2.3.1.3), including 401/403
		{"403 as on wwwassets.unipd.it", 403, "Forbidden", nil, false, "/sites/default/files/2026-07/x.pdf", true},
		{"401", 401, "", nil, false, "/x", true},
		{"404", 404, "", nil, false, "/x", true},
		{"410", 410, "", nil, false, "/x", true},
		// temporary: do not crawl, retry later
		{"429 rate limited", 429, "", nil, true, "", false},
		{"500", 500, "", nil, true, "", false},
		{"503", 503, "", nil, true, "", false},
		{"network error", 0, "", errors.New("connection reset"), true, "", false},
		{"unfollowed redirect", 302, "", nil, true, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := RobotsFromResponse(tt.status, []byte(tt.body), tt.fetchErr)
			if tt.wantErr {
				if !errors.Is(err, ErrRobotsUnavailable) {
					t.Fatalf("err = %v, want ErrRobotsUnavailable", err)
				}
				if r.Allowed(ua, "/anything") {
					t.Error("a nil policy must not allow crawling")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := r.Allowed(ua, tt.path); got != tt.wantAllow {
				t.Errorf("Allowed(%q) = %v, want %v", tt.path, got, tt.wantAllow)
			}
		})
	}
}

// Characterisation test: documents how github.com/temoto/robotstxt (v1.1.2)
// maps statuses itself. We do not rely on it (RobotsFromResponse maps them
// explicitly), but if an upgrade changes this, we want to notice and re-read
// the library's release notes.
func TestLibraryStatusMappingCharacterisation(t *testing.T) {
	for _, status := range []int{401, 403, 404} {
		d, err := robotstxt.FromStatusAndBytes(status, nil)
		if err != nil || !d.TestAgent("/x", "UniPDAssistantBot") {
			t.Errorf("library: status %d should allow all (err=%v)", status, err)
		}
	}
	d, err := robotstxt.FromStatusAndBytes(503, nil)
	if err != nil || d.TestAgent("/x", "UniPDAssistantBot") {
		t.Errorf("library: status 503 should disallow all (err=%v)", err)
	}
}
