package svg

import (
	"fmt"
	"html"
	"strings"

	"github.com/hyperfinitism/gh-contrib-stats/internal/config"
	"github.com/hyperfinitism/gh-contrib-stats/internal/github"
)

// formatNumber adds comma separators to integers (e.g. 1234567 → "1,234,567").
func formatNumber(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	offset := len(s) % 3
	if offset > 0 {
		b.WriteString(s[:offset])
	}
	for i := offset; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

type statEntry struct {
	Key   string // icon key (e.g. "pr", "commit", …)
	Label string
	Value int
}

func labelFor(key string) string {
	switch key {
	case "pr":
		return "Pull Requests"
	case "commit":
		return "Commits"
	case "issue":
		return "Issues"
	case "review":
		return "Reviews"
	case "discussion":
		return "Discussions"
	default:
		return key
	}
}

func statValue(stats *github.ContributionStats, key string) int {
	switch key {
	case "pr":
		return stats.PR
	case "commit":
		return stats.Commit
	case "issue":
		return stats.Issue
	case "review":
		return stats.Review
	case "discussion":
		return stats.Discussion
	default:
		return 0
	}
}

func showEnabled(show *config.ResolvedShow, key string) bool {
	switch key {
	case "pr":
		return show.PR
	case "commit":
		return show.Commit
	case "issue":
		return show.Issue
	case "review":
		return show.Review
	case "discussion":
		return show.Discussion
	default:
		return false
	}
}

func weightValue(w *config.ResolvedWeight, key string) uint {
	switch key {
	case "pr":
		return w.PR
	case "commit":
		return w.Commit
	case "issue":
		return w.Issue
	case "review":
		return w.Review
	case "discussion":
		return w.Discussion
	default:
		return 0
	}
}

func selectEnabled(sel *config.ResolvedSelect, key string) bool {
	switch key {
	case "pr":
		return sel.PR
	case "commit":
		return sel.Commit
	case "issue":
		return sel.Issue
	case "review":
		return sel.Review
	case "discussion":
		return sel.Discussion
	default:
		return false
	}
}

// Generate creates a GitHub-readme-stats style SVG card.
func Generate(cfg *config.ResolvedConfig, stats *github.ContributionStats) string {
	var b strings.Builder
	t := cfg.Theme

	// Determine which stats to show.
	allKeys := []string{"pr", "commit", "issue", "review", "discussion"}
	var shownStats []statEntry
	for _, key := range allKeys {
		if showEnabled(&cfg.Show, key) {
			shownStats = append(shownStats, statEntry{
				Key:   key,
				Label: labelFor(key),
				Value: statValue(stats, key),
			})
		}
	}

	// Calculate total score.
	totalScore := 0
	for _, key := range allKeys {
		if selectEnabled(&cfg.Select, key) {
			totalScore += statValue(stats, key) * int(weightValue(&cfg.Weight, key))
		}
	}

	// Compute card dimensions.
	cardWidth := 495
	iconTextIndent := 22 // horizontal offset for text after a 16px icon
	padding := 25
	headerHeight := 45
	scoreHeight := 35
	statLineHeight := 25
	statsHeight := len(shownStats) * statLineHeight
	repoHeaderHeight := 0
	repoLineHeight := 22
	reposHeight := 0
	if len(stats.TopRepos) > 0 {
		repoHeaderHeight = 35
		reposHeight = len(stats.TopRepos) * repoLineHeight
	}
	cardHeight := padding + headerHeight + scoreHeight + statsHeight + repoHeaderHeight + reposHeight + padding

	// Start SVG.
	b.WriteString(fmt.Sprintf(`<svg width="%d" height="%d" viewBox="0 0 %d %d" xmlns="http://www.w3.org/2000/svg">`, cardWidth, cardHeight, cardWidth, cardHeight))
	b.WriteString("\n")

	// Styles — colors driven by the resolved theme.
	b.WriteString(`<style>
    .header { font: 600 18px 'Segoe UI', Ubuntu, Sans-Serif; fill: ` + t.Title + `; }
    .stat-label { font: 400 14px 'Segoe UI', Ubuntu, Sans-Serif; fill: ` + t.Text + `; }
    .stat-value { font: 700 14px 'Segoe UI', Ubuntu, Sans-Serif; fill: ` + t.Text + `; }
    .score-label { font: 600 16px 'Segoe UI', Ubuntu, Sans-Serif; fill: ` + t.Score + `; }
    .score-value { font: 800 16px 'Segoe UI', Ubuntu, Sans-Serif; fill: ` + t.Score + `; }
    .repo-header { font: 600 13px 'Segoe UI', Ubuntu, Sans-Serif; fill: ` + t.Muted + `; }
    .repo-name { font: 400 12px 'Segoe UI', Ubuntu, Sans-Serif; fill: ` + t.Title + `; }
    .repo-score { font: 400 12px 'Segoe UI', Ubuntu, Sans-Serif; fill: ` + t.Muted + `; }
    @keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }
    .fade-in { animation: fadeIn 0.3s ease-in-out forwards; }
  </style>`)
	b.WriteString("\n")

	// Background.
	b.WriteString(fmt.Sprintf(`<rect x="0.5" y="0.5" rx="4.5" width="%d" height="%d" fill="%s" stroke="%s"/>`,
		cardWidth-1, cardHeight-1, t.Background, t.Border))
	b.WriteString("\n")

	// Content group.
	b.WriteString(fmt.Sprintf(`<g transform="translate(%d, %d)">`, padding, padding))
	b.WriteString("\n")

	// Header.
	y := 0
	b.WriteString(fmt.Sprintf(`<text x="0" y="%d" class="header">%s's Contribution Stats</text>`, y+20, html.EscapeString(cfg.Username)))
	b.WriteString("\n")
	y += headerHeight

	// Score.
	b.WriteString(fmt.Sprintf(`<g transform="translate(0, %d)" class="fade-in">`, y))
	b.WriteString(iconColorized("score", t.Score))
	b.WriteString(fmt.Sprintf(`<text x="%d" y="13" class="score-label">Total Score:</text>`, iconTextIndent))
	b.WriteString(fmt.Sprintf(`<text x="%d" y="13" class="score-value" text-anchor="end">%s</text>`, cardWidth-2*padding, formatNumber(totalScore)))
	b.WriteString(`</g>`)
	b.WriteString("\n")
	y += scoreHeight

	// Stats.
	for i, s := range shownStats {
		statY := y + i*statLineHeight
		delay := float64(i) * 0.15
		b.WriteString(fmt.Sprintf(`<g transform="translate(0, %d)" class="fade-in" style="animation-delay: %.1fs;">`, statY, delay))

		// Icon — loaded from the embedded SVG files and colorized per theme.
		b.WriteString(iconColorized(s.Key, t.IconColor(s.Key)))

		// Label.
		b.WriteString(fmt.Sprintf(`<text x="%d" y="13" class="stat-label">%s:</text>`, iconTextIndent, s.Label))

		// Value (right-aligned).
		b.WriteString(fmt.Sprintf(`<text x="%d" y="13" class="stat-value" text-anchor="end">%s</text>`, cardWidth-2*padding, formatNumber(s.Value)))

		b.WriteString(`</g>`)
		b.WriteString("\n")
	}

	y += statsHeight

	// Top repos.
	if len(stats.TopRepos) > 0 {
		y += 10
		b.WriteString(fmt.Sprintf(`<g transform="translate(0, %d)">`, y))
		b.WriteString(iconColorized("repo", t.Muted))
		b.WriteString(fmt.Sprintf(`<text x="%d" y="13" class="repo-header">Top Contributed Repositories</text>`, iconTextIndent))
		b.WriteString(`</g>`)
		b.WriteString("\n")
		y += repoHeaderHeight - 10

		for i, repo := range stats.TopRepos {
			repoY := y + i*repoLineHeight
			delay := float64(len(shownStats)+i) * 0.15
			b.WriteString(fmt.Sprintf(`<g transform="translate(%d, %d)" class="fade-in" style="animation-delay: %.1fs;">`, iconTextIndent, repoY, delay))
			b.WriteString(fmt.Sprintf(`<text x="0" y="13" class="repo-name">%d. %s</text>`, i+1, html.EscapeString(repo.Name)))
			b.WriteString(fmt.Sprintf(`<text x="%d" y="13" class="repo-score" text-anchor="end">%s pts</text>`, cardWidth-2*padding-iconTextIndent, formatNumber(repo.WeightedScore)))
			b.WriteString(`</g>`)
			b.WriteString("\n")
		}
	}

	b.WriteString(`</g>`)
	b.WriteString("\n")
	b.WriteString(`</svg>`)

	return b.String()
}
