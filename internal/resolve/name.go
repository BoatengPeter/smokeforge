package resolve

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const maxProjectNameLength = 64

// projectNamePattern allows letters and digits in any script, plus spaces,
// dots, underscores, and hyphens inside the name. The project name is
// rendered verbatim into source strings, JSON, YAML, shell scripts, and
// desktop entries, so quotes, backslashes, `$`, backticks, slashes, angle
// brackets, and control characters are rejected rather than escaped.
var projectNamePattern = regexp.MustCompile(`^[\p{L}\p{N}](?:[\p{L}\p{N} ._-]*[\p{L}\p{N}.])?$`)

// ValidateProjectName reports whether name is safe to use as a new project's
// display name.
func ValidateProjectName(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("project name is required")
	}

	if len([]rune(name)) > maxProjectNameLength {
		return fmt.Errorf("project name must be at most %d characters", maxProjectNameLength)
	}

	if !projectNamePattern.MatchString(name) {
		return fmt.Errorf(
			"project name %q may only contain letters, digits, spaces, '.', '_', and '-', "+
				"and must start with a letter or digit",
			name,
		)
	}

	return nil
}
