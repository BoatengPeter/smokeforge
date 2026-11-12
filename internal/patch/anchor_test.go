package patch

import (
	"strings"
	"testing"
)

const goBase = "package main\n\nimport (\n\t\"fmt\"\n\t// smokeforge:anchor imports\n)\n"

func TestInsertBlockAfter(t *testing.T) {
	t.Parallel()

	got, err := insertBlock(insertRequest{
		content: goBase,
		anchor:  "imports",
		marker:  "logging:basic",
		block:   `"example.com/app/internal/log"`,
	})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	wantLines := []string{
		"\t// smokeforge:begin logging:basic",
		"\t\"example.com/app/internal/log\"",
		"\t// smokeforge:end logging:basic",
	}
	for _, line := range wantLines {
		if !strings.Contains(got, line) {
			t.Fatalf("output missing %q:\n%s", line, got)
		}
	}
}

func TestInsertBlockBefore(t *testing.T) {
	t.Parallel()

	got, err := insertBlock(insertRequest{
		content: goBase,
		anchor:  "imports",
		marker:  "before",
		block:   `"os"`,
		before:  true,
	})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	begin := strings.Index(got, "smokeforge:begin before")

	anchor := strings.Index(got, "smokeforge:anchor imports")
	if begin == -1 || anchor == -1 || begin > anchor {
		t.Fatalf("block was not inserted before the anchor:\n%s", got)
	}
}

func TestInsertBlockCommentStyles(t *testing.T) {
	t.Parallel()

	got, err := insertBlock(insertRequest{
		file:    "main.py",
		content: "# smokeforge:anchor imports\n",
		anchor:  "imports",
		marker:  "py",
		block:   "import os",
	})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	if !strings.Contains(got, "# smokeforge:begin py") {
		t.Fatalf("python markers use the wrong comment style:\n%s", got)
	}
}

func TestInsertBlockErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "missing anchor",
			content: "package main\n",
			want:    "anchor",
		},
		{
			name:    "duplicate anchor",
			content: "// smokeforge:anchor imports\n// smokeforge:anchor imports\n",
			want:    "exactly once",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := insertBlock(insertRequest{
				content: tt.content,
				anchor:  "imports",
				marker:  "x",
				block:   "y",
			})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestInsertBlockIdempotent(t *testing.T) {
	t.Parallel()

	req := insertRequest{
		content: goBase,
		anchor:  "imports",
		marker:  "logging",
		block:   `"os"`,
	}

	first, err := insertBlock(req)
	if err != nil {
		t.Fatal(err)
	}

	req.content = first

	second, err := insertBlock(req)
	if err != nil {
		t.Fatal(err)
	}

	if first != second {
		t.Fatalf("second insertion changed the file:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}
