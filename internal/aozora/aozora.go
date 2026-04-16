package aozora

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)


var (
	// Single-pass ruby: group 1,2 = pipe ruby base/reading; group 3,4 = non-pipe (kanji only)
	reAllRuby = regexp.MustCompile(`｜([^《\n]+)《([^》\n]+)》|(\p{Han}+)《([^》\n]+)》`)

	// ［＃改ページ］
	rePageBreak = regexp.MustCompile(`［＃改ページ］`)

	// ［＃...］ 全般（改ページ以外）
	reAnnotation = regexp.MustCompile(`［＃[^］]*］`)

	// ヘッダー部区切り
	reSeparator = regexp.MustCompile(`-{5,}`)
)

// Chapter represents a heading found in the text.
type Chapter struct {
	Title    string
	Anchor   string // id attribute in the XHTML (e.g. "ch1")
}

// Parse converts raw Aozora text to HTML paragraphs.
// Returns title, author, paragraphs, and detected chapters.
func Parse(raw string) (title, author string, paragraphs []string, chapters []Chapter) {
	lines := strings.Split(normalizeNewlines(raw), "\n")
	lines, title, author = stripHeader(lines)
	paragraphs, chapters = buildParagraphs(lines)
	return
}

func normalizeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}

// stripHeader removes the Aozora header (title/author/separator lines at top
// and colophon at bottom), returning remaining lines plus extracted metadata.
func stripHeader(lines []string) ([]string, string, string) {
	// Find first separator line (--------)
	sepIdx := -1
	for i, l := range lines {
		if reSeparator.MatchString(l) {
			sepIdx = i
			break
		}
	}

	title, author := "", ""
	if sepIdx >= 1 {
		title = strings.TrimSpace(lines[0])
	}
	if sepIdx >= 2 {
		author = strings.TrimSpace(lines[1])
	}

	start := 0
	if sepIdx >= 0 {
		start = sepIdx + 1
	}
	// Skip blank lines after separator
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}

	// If annotation block exists, skip to the second separator
	if start < len(lines) && strings.Contains(lines[start], "テキスト中に現れる記号について") {
		for i := start + 1; i < len(lines); i++ {
			if reSeparator.MatchString(lines[i]) {
				start = i + 1
				break
			}
		}
		// Skip blank lines after second separator
		for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
			start++
		}
	}

	// Find trailing colophon separator: a separator line followed by 底本 content.
	end := len(lines)
	for i := len(lines) - 1; i > start; i-- {
		if reSeparator.MatchString(lines[i]) {
			// Verify it's actually the colophon by checking nearby lines.
			for j := i + 1; j < len(lines) && j < i+5; j++ {
				if strings.Contains(lines[j], "底本") {
					end = i
					break
				}
			}
			if end != len(lines) {
				break
			}
		}
	}

	return lines[start:end], title, author
}

func buildParagraphs(lines []string) ([]string, []Chapter) {
	var result []string
	var chapters []Chapter
	var buf []string
	chapterIdx := 0

	flush := func() {
		if len(buf) > 0 {
			joined := strings.Join(buf, "")
			p := convertLine(joined)
			if p != "" {
				result = append(result, fmt.Sprintf("<p>%s</p>", p))
			}
			buf = nil
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if rePageBreak.MatchString(trimmed) {
			flush()
			result = append(result, `<hr class="pagebreak"/>`)
			continue
		}

		if trimmed == "" {
			flush()
			continue
		}

		if isHeading(trimmed) {
			flush()
			chapterIdx++
			anchor := fmt.Sprintf("ch%d", chapterIdx)
			text := convertLine(trimmed)
			result = append(result, fmt.Sprintf(`<h2 id="%s">%s</h2>`, anchor, text))
			chapters = append(chapters, Chapter{Title: stripTags(text), Anchor: anchor})
			continue
		}

		buf = append(buf, trimmed)
	}
	flush()
	return result, chapters
}

var reTag = regexp.MustCompile(`<[^>]+>`)

func stripTags(s string) string {
	return reTag.ReplaceAllString(s, "")
}

func isHeading(s string) bool {
	// Lines starting with 「一」「二」etc. or short lines wrapped in 【】
	if len([]rune(s)) <= 20 && (strings.HasPrefix(s, "【") || strings.HasPrefix(s, "第")) {
		return true
	}
	return false
}

func convertLine(s string) string {
	// Strip annotations first
	s = reAnnotation.ReplaceAllString(s, "")

	// Single-pass ruby conversion with correct HTML escaping.
	// Plain text segments between ruby spans are HTML-escaped individually.
	var result strings.Builder
	lastIdx := 0
	for _, loc := range reAllRuby.FindAllStringSubmatchIndex(s, -1) {
		result.WriteString(html.EscapeString(s[lastIdx:loc[0]]))
		var base, reading string
		if loc[2] >= 0 {
			// pipe ruby
			base = s[loc[2]:loc[3]]
			reading = s[loc[4]:loc[5]]
		} else {
			// non-pipe ruby
			base = s[loc[6]:loc[7]]
			reading = s[loc[8]:loc[9]]
		}
		fmt.Fprintf(&result, "<ruby>%s<rt>%s</rt></ruby>",
			html.EscapeString(base), html.EscapeString(reading))
		lastIdx = loc[1]
	}
	result.WriteString(html.EscapeString(s[lastIdx:]))

	return strings.TrimSpace(result.String())
}
