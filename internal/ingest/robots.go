package ingest

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/temoto/robotstxt"
)

// ErrRobotsUnavailable means robots.txt could not be obtained for a reason that
// is expected to be temporary. The host must not be crawled until a later retry
// succeeds.
var ErrRobotsUnavailable = errors.New("robots.txt temporarily unavailable")

// Robots is the crawl policy of one host.
type Robots struct {
	data     *robotstxt.RobotsData // nil = no restrictions
	allowAll bool
}

// RobotsFromResponse turns the outcome of fetching /robots.txt into a policy
// following RFC 9309 §2.3.1:
//
//   - 2xx: parse the file.
//   - any 4xx ("unavailable"): the file does not exist or is not for us, so there
//     are no restrictions. This includes 401 and 403. (wwwassets.unipd.it answers
//     403 and is therefore crawlable; its X-Robots-Tag is handled separately.)
//   - 5xx, 429, network error, or an unexpected status: treated as disallow for
//     now. ErrRobotsUnavailable is returned so the caller retries later with
//     backoff instead of caching a permanent "disallow all".
//
// 429 is a 4xx, which RFC 9309 would allow; we deliberately treat it as
// temporary because it is a rate-limit signal.
//
// The status mapping is explicit here, not delegated to the library, so that a
// library upgrade cannot change crawl behaviour (see robots_test.go).
func RobotsFromResponse(status int, body []byte, fetchErr error) (*Robots, error) {
	if fetchErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrRobotsUnavailable, fetchErr)
	}
	switch {
	case status >= 200 && status < 300:
		d, err := robotstxt.FromBytes(body)
		if err != nil {
			return nil, fmt.Errorf("%w: parse: %w", ErrRobotsUnavailable, err)
		}
		return &Robots{data: d}, nil
	case status == http.StatusTooManyRequests:
		return nil, fmt.Errorf("%w: status %d", ErrRobotsUnavailable, status)
	case status >= 400 && status < 500:
		return &Robots{allowAll: true}, nil
	default: // 1xx, unfollowed 3xx, 5xx
		return nil, fmt.Errorf("%w: status %d", ErrRobotsUnavailable, status)
	}
}

// Allowed reports whether userAgent (the product token, e.g. "UniPDAssistantBot")
// may fetch path (path and query, as in the request line).
func (r *Robots) Allowed(userAgent, path string) bool {
	if r == nil || r.allowAll || r.data == nil {
		return r != nil
	}
	return r.data.TestAgent(path, userAgent)
}
