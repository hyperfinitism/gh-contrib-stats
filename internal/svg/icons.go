package svg

import (
	"fmt"
	"os"
	"strings"
	"sync"

	octicons "github.com/hyperfinitism/gh-contrib-stats/third_party"
)

// iconMeta maps application-level icon keys to their upstream SVG filenames.
var iconMeta = map[string]string{
	"pr":         "git-pull-request-16.svg",
	"commit":     "git-commit-16.svg",
	"issue":      "issue-opened-16.svg",
	"review":     "code-review-16.svg",
	"discussion": "comment-discussion-16.svg",
	"star":       "star-fill-16.svg",
	"repo":       "repo-16.svg",
}

// iconCache stores the parsed inner SVG content (the <path> elements between
// the outer <svg> wrapper tags) so each file is read from the embed.FS at
// most once.
var (
	iconCache   = make(map[string]string)
	iconCacheMu sync.RWMutex
)

// loadIconInner reads an SVG file from the embedded third-party FS and
// returns its inner content (the <path .../> elements) with the outer
// <svg>…</svg> wrapper stripped.
func loadIconInner(filename string) (string, error) {
	iconCacheMu.RLock()
	inner, ok := iconCache[filename]
	iconCacheMu.RUnlock()
	if ok {
		return inner, nil
	}

	data, err := octicons.FS.ReadFile("primer-octicons/" + filename)
	if err != nil {
		return "", fmt.Errorf("reading embedded icon %s: %w", filename, err)
	}

	inner = extractInnerSVG(string(data))

	iconCacheMu.Lock()
	iconCache[filename] = inner
	iconCacheMu.Unlock()

	return inner, nil
}

// extractInnerSVG returns everything between the first '>' of the opening
// <svg …> tag and the closing </svg> tag.
//
// This is intentionally a simple string-based extraction rather than a full
// XML parse — the input files are trusted, machine-generated Octicon SVGs
// with a known, stable single-line structure:
//
//	<svg xmlns="…" width="16" height="16" viewBox="0 0 16 16"><path d="…"/></svg>
func extractInnerSVG(raw string) string {
	// Find end of opening <svg ...> tag.
	start := strings.Index(raw, ">")
	if start == -1 {
		return raw
	}
	start++ // skip past '>'

	// Find closing </svg>.
	end := strings.LastIndex(raw, "</svg>")
	if end == -1 || end <= start {
		return raw[start:]
	}

	return raw[start:end]
}

// iconInner returns the cached inner SVG content for the given app-level key.
func iconInner(key string) (string, error) {
	file, ok := iconMeta[key]
	if !ok {
		return "", fmt.Errorf("unknown icon key: %q", key)
	}
	return loadIconInner(file)
}

// iconColorized renders a 16×16 inline <svg> element with the inner paths
// filled in the given colour, ready to be placed inside the card markup.
func iconColorized(key, color string) string {
	inner, err := iconInner(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: icon %q: %v\n", key, err)
		return ""
	}
	return fmt.Sprintf(
		`<svg x="0" y="0" width="16" height="16" viewBox="0 0 16 16" fill="%s">%s</svg>`,
		color, inner,
	)
}