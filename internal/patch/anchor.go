package patch

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

type insertRequest struct {
	file    string
	content string
	anchor  string
	marker  string
	block   string
	before  bool
}

func insertBlock(req insertRequest) (string, error) {
	if strings.TrimSpace(req.anchor) == "" {
		return "", errors.New("anchor must not be empty")
	}

	if strings.TrimSpace(req.marker) == "" {
		return "", errors.New("marker must not be empty")
	}

	if hasMarkerLine(req.content, "smokeforge:begin ", req.marker) {
		return req.content, nil
	}

	lines := strings.Split(req.content, "\n")
	matches := make([]int, 0, 1)

	for index, line := range lines {
		if token, ok := lineToken(line, "smokeforge:anchor "); ok && token == req.anchor {
			matches = append(matches, index)
		}
	}

	if len(matches) != 1 {
		return "", fmt.Errorf(
			"anchor %q must appear exactly once in %q (found %d)",
			req.anchor,
			req.file,
			len(matches),
		)
	}

	anchorIndex := matches[0]
	indent := leadingWhitespace(lines[anchorIndex])
	prefix, suffix := commentStyleForAnchor(req.file, lines[anchorIndex])

	blockLines := []string{
		indent + markerLine(prefix, suffix, "smokeforge:begin "+req.marker),
	}
	if req.block != "" {
		for line := range strings.SplitSeq(strings.TrimSuffix(req.block, "\n"), "\n") {
			if line == "" {
				blockLines = append(blockLines, "")

				continue
			}

			blockLines = append(blockLines, indent+line)
		}
	}

	blockLines = append(blockLines, indent+markerLine(prefix, suffix, "smokeforge:end "+req.marker))

	insertAt := anchorIndex + 1
	if req.before {
		insertAt = anchorIndex
	}

	result := make([]string, 0, len(lines)+len(blockLines))
	result = append(result, lines[:insertAt]...)
	result = append(result, blockLines...)
	result = append(result, lines[insertAt:]...)

	return strings.Join(result, "\n"), nil
}

func hasMarkerLine(content, prefix, marker string) bool {
	for line := range strings.SplitSeq(content, "\n") {
		if token, ok := lineToken(line, prefix); ok && token == marker {
			return true
		}
	}

	return false
}

func lineToken(line, prefix string) (string, bool) {
	_, rest, found := strings.Cut(line, prefix)
	if !found {
		return "", false
	}

	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return "", true
	}

	return fields[0], true
}

func leadingWhitespace(line string) string {
	for index, r := range line {
		if r != ' ' && r != '\t' {
			return line[:index]
		}
	}

	return line
}

func markerLine(prefix, suffix, token string) string {
	if suffix == "" {
		return prefix + " " + token
	}

	return prefix + " " + token + " " + suffix
}

func commentStyleForAnchor(file, anchorLine string) (prefix, suffix string) {
	trimmed := strings.TrimSpace(anchorLine)

	switch {
	case strings.HasPrefix(trimmed, "{/*"):
		return "{/*", "*/}"
	case strings.HasPrefix(trimmed, "<!--"):
		return "<!--", "-->"
	default:
		return commentStyle(file)
	}
}

func commentStyle(file string) (prefix, suffix string) {
	base := strings.ToLower(filepath.Base(file))
	extension := strings.ToLower(filepath.Ext(base))

	switch base {
	case "makefile", "dockerfile", ".gitignore", ".env", ".env.example":
		return "#", ""
	}

	switch extension {
	case ".go", ".rs", ".js", ".jsx", ".ts", ".tsx", ".kt", ".kts", ".swift",
		".java", ".c", ".h", ".cc", ".cpp", ".cs", ".scala", ".dart":
		return "//", ""
	case ".py", ".rb", ".sh", ".bash", ".zsh", ".yaml", ".yml", ".toml",
		".tf", ".conf", ".ini":
		return "#", ""
	case ".sql":
		return "--", ""
	case ".html", ".htm", ".xml", ".md", ".svg":
		return "<!--", "-->"
	default:
		return "//", ""
	}
}
