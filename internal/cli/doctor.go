package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

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
