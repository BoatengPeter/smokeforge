package manifest

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
)

// ErrCLITooOld reports that a registry declares a minCliVersion newer than the
// running smokeforge binary.
var ErrCLITooOld = errors.New("smokeforge is too old for this registry")

// CheckCompatibility reads only minCliVersion from raw manifest data and
// reports whether cliVersion may load it.
//
// It decodes leniently on purpose: it runs before the strict parse, so an
// older CLI facing a registry with fields it does not know gets an upgrade
// message instead of a YAML "field not found" error. Development builds
// ("dev", "(devel)", or any non-semver version) skip the check.
func CheckCompatibility(data []byte, cliVersion string) error {
	var header struct {
		MinCLIVersion string `yaml:"minCliVersion"`
	}

	// Lenient decode: unknown fields are ignored here.
	if err := yaml.Unmarshal(data, &header); err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}

	if header.MinCLIVersion == "" {
		return nil
	}

	required, err := canonicalVersion(header.MinCLIVersion)
	if err != nil {
		return err
	}

	current := semver.Canonical(withV(strings.TrimSpace(cliVersion)))
	if current == "" {
		return nil
	}

	if semver.Compare(current, required) >= 0 {
		return nil
	}

	return fmt.Errorf(
		"%w: it requires smokeforge %s or newer, but this is %s; "+
			"run `smokeforge update` to upgrade",
		ErrCLITooOld,
		strings.TrimPrefix(required, "v"),
		strings.TrimPrefix(current, "v"),
	)
}

func canonicalVersion(value string) (string, error) {
	canonical := semver.Canonical(withV(strings.TrimSpace(value)))
	if canonical == "" {
		return "", fmt.Errorf("minCliVersion %q must be a semantic version such as 0.4.0", value)
	}

	return canonical, nil
}

func withV(value string) string {
	if value == "" || strings.HasPrefix(value, "v") {
		return value
	}

	return "v" + value
}
