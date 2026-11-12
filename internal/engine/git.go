package engine

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func initGit(ctx context.Context, dir string) error {
	command := exec.CommandContext(ctx, "git", "init", "--quiet")
	command.Dir = dir

	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"initialize git repository: %w: %s",
			err,
			strings.TrimSpace(string(output)),
		)
	}

	return nil
}
