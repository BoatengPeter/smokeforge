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

	if strings.Contains(req.content, "smokeforge:begin "+req.marker) {
		return req.content, nil
	}

	lines := strings.Split(req.content, "\n")
	anchorToken := "smokeforge:anchor " + req.anchor

	matches := make([]int, 0, 1)

	for index, line := range lines {
		if strings.Contains(line, anchorToken) {
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
	prefix, suffix := commentStyle(req.file)

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
