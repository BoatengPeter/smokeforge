package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/BoatengPeter/smokeforge/internal/project"
	"github.com/BoatengPeter/smokeforge/internal/registry"
	"github.com/spf13/cobra"
)

type doctorCheck struct {
	name     string
	checkErr error
}

func newDoctorCmd(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check the local environment and registry",
		RunE: func(cmd *cobra.Command, _ []string) error {
			checks := []doctorCheck{
				{name: "registry", checkErr: checkRegistry(cmd, opts)},
				{name: "cache", checkErr: checkCache(opts.cacheDir)},
				{name: "git", checkErr: checkGit()},
			}
			if projectCheck, ok := checkProject(); ok {
				checks = append(checks, projectCheck)
			}

			problems := make([]error, 0, len(checks))
			for _, check := range checks {
				if err := reportStatus(opts.out, check); err != nil {
					problems = append(problems, err)
				}
			}

			if len(problems) > 0 {
				return errors.Join(problems...)
			}

			return nil
		},
	}
}

func checkRegistry(cmd *cobra.Command, opts *options) error {
	_, err := loadRegistry(cmd.Context(), opts)

	return err
}

func checkCache(dir string) error {
	if dir == "" {
		defaultDir, err := registry.DefaultCacheDir()
		if err != nil {
			return err
		}

		dir = defaultDir
	}

	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create cache directory: %w", err)
	}

	file, err := os.CreateTemp(dir, ".doctor-*")
	if err != nil {
		return fmt.Errorf("write to cache directory: %w", err)
	}

	name := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(name)

		return fmt.Errorf("close cache probe: %w", err)
	}

	if err := os.Remove(name); err != nil {
		return fmt.Errorf("remove cache probe: %w", err)
	}

	return nil
}

func checkGit() error {
	_, err := exec.LookPath("git")

	return err
}

func reportStatus(out io.Writer, check doctorCheck) error {
	if check.checkErr != nil {
		if err := writef(out, "%s: fail (%v)\n", check.name, check.checkErr); err != nil {
			return err
		}

		return fmt.Errorf("%s check failed: %w", check.name, check.checkErr)
	}

	return writef(out, "%s: ok\n", check.name)
}

func checkProject() (doctorCheck, bool) {
	root, err := project.FindRoot(".")
	if errors.Is(err, project.ErrNotProject) {
		return doctorCheck{}, false
	}

	if err != nil {
		return doctorCheck{name: "project", checkErr: err}, true
	}

	state, err := project.LoadState(root)
	if err != nil {
		return doctorCheck{name: "project", checkErr: err}, true
	}

	if _, err := project.LoadLock(root); err != nil {
		return doctorCheck{name: "project", checkErr: err}, true
	}

	drifts, err := project.CheckDrift(root, state)
	if err != nil {
		return doctorCheck{name: "project", checkErr: err}, true
	}

	if len(drifts) > 0 {
		paths := make([]string, 0, len(drifts))
		for _, drift := range drifts {
			paths = append(paths, drift.Path)
		}

		return doctorCheck{
			name:     "project",
			checkErr: fmt.Errorf("%d file(s) drifted: %s", len(drifts), strings.Join(paths, ", ")),
		}, true
	}

	return doctorCheck{name: "project"}, true
}
