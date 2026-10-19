// Package ingest defines the crawl → parse → chunk pipeline contracts.
// Implementations arrive in M2. Each stage is an interface so it can be
// tested against saved fixtures without the network.
package ingest

import (
	"context"
	"time"
)

type DocType string

const (
	DocBando  DocType = "bando"
	DocAvviso DocType = "avviso"
	DocFAQ    DocType = "faq"
	DocPage   DocType = "page"
	DocOther  DocType = "other"
)

// FetchResult is one HTTP(S) fetch. NotModified is true on a 304 or when the
// raw hash equals the previous one; Body is then nil.
type FetchResult struct {
	URL          string // final URL after redirects
	Status       int
	ContentType  string
	Body         []byte
	ETag         string
	LastModified *time.Time
	// XRobotsTag is the raw X-Robots-Tag header ("noindex, nofollow" on
	// wwwassets.unipd.it PDFs). It is stored on the version and "nofollow" is
	// enforced: no links are extracted from such responses.
	XRobotsTag  string
	FetchedAt   time.Time
	NotModified bool
}

// Fetcher enforces robots.txt, per-host rate limits and conditional requests.
// It returns ErrBlockedByRobots instead of fetching disallowed URLs.
type Fetcher interface {
	Fetch(ctx context.Context, url string, prev *Validators) (*FetchResult, error)
}

// Validators are the conditional-request values from the previous fetch.
type Validators struct {
	ETag         string
	LastModified *time.Time
}

// Section is one structural unit of a parsed document: an "Art. N" of a bando,
// a heading block of an HTML page, a FAQ entry.
type Section struct {
	Path     string // "Art. 7" or "Art. 7 > comma 2" or "Requisiti > Merito"
	Heading  string
	Text     string
	PageFrom int // 0 for HTML
	PageTo   int
}

// ParsedDocument is the output of a parser, before chunking.
type ParsedDocument struct {
	Title        string
	Language     string
	DocType      DocType
	AcademicYear string     // "2026/27" or "" if unknown
	PublishedAt  *time.Time // date printed on the document, if found
	// PublishedAtSource is "document" when the date was read from the text, or
	// "http_last_modified" when the document has no date of its own and the
	// Last-Modified header is used as a fallback (maps to document_versions.published_at_source).
	PublishedAtSource string
	Text              string // normalised main content; hashed for change detection
	Sections          []Section
	// Links feeds the frontier. PDF parsers MUST leave it nil: links are never
	// followed out of PDFs (they are served with X-Robots-Tag nofollow, and new
	// corrections are discovered from HTML listing pages and the sitemap instead).
	Links    []string
	NeedsOCR bool // PDF with no text layer
}

// Parser handles one content type (HTML, PDF).
type Parser interface {
	Accepts(contentType string) bool
	Parse(ctx context.Context, res *FetchResult) (*ParsedDocument, error)
}

// Chunk is a retrievable unit. Chunking follows document structure; a Section
// longer than the model limit is split on paragraph boundaries, and every
// piece keeps the section's Path.
type Chunk struct {
	Ordinal     int
	SectionPath string
	Heading     string
	Content     string
	PageFrom    int
	PageTo      int
	TokenCount  int
}

type Chunker interface {
	Chunk(doc *ParsedDocument) []Chunk
}

// ManualInput describes a file the operator downloaded by hand, for documents
// a site blocks for bots. It goes through the same parse → version → chunk →
// index pipeline as a crawled document; citations point at SourceURL, never at
// the local file. Versions created this way have ingest_source = 'manual'.
type ManualInput struct {
	SourceURL     string // original public URL; becomes documents.canonical_url
	FilePath      string // local file with the bytes
	ContentType   string // "" = sniff from the file
	DocType       DocType
	Language      string // "it", "en"
	TranslationOf string // optional: canonical URL of the Italian original
}
