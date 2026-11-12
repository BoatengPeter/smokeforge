package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/huh"
)

func TestFilterFeatureValues(t *testing.T) {
	t.Parallel()

	options := []huh.Option[string]{
		huh.NewOption("Auth", "auth"),
		huh.NewOption("Database", "database:postgres"),
	}

	got := filterFeatureValues(
		[]string{"auth", "missing", "database:postgres"},
		options,
	)

	if strings.Join(got, ",") != "auth,database:postgres" {
		t.Fatalf("filtered = %v", got)
	}
}
