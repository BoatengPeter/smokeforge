package cli

import (
	"strings"
	"testing"
)

func TestRootHelp(t *testing.T) {
	t.Parallel()

	out, _, err := executeCommand(t, "--help")
	if err != nil {
		t.Fatalf("help: %v", err)
	}

	for _, want := range []string{
		"smokeforge",
		"new",
		"add",
		"remove",
		"completion",
		"https://github.com/BoatengPeter/smokeforge",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("help output missing %q:\n%s", want, out)
		}
	}
}

func TestNewHelp(t *testing.T) {
	t.Parallel()

	out, _, err := executeCommand(t, "new", "--help")
	if err != nil {
		t.Fatalf("help: %v", err)
	}

	for _, want := range []string{
		"--preset",
		"fullstack",
		"frontend",
		"api",
		"api-custom",
		"Examples:",
		"smokeforge new my-app",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("new help missing %q:\n%s", want, out)
		}
	}

	if strings.Contains(out, "Production") {
		t.Fatalf("new help should not label a preset as production:\n%s", out)
	}
}

func TestHelpSubcommand(t *testing.T) {
	t.Parallel()

	out, _, err := executeCommand(t, "help", "new")
	if err != nil {
		t.Fatalf("help new: %v", err)
	}

	if !strings.Contains(out, "smokeforge new my-app") {
		t.Fatalf("help new missing example:\n%s", out)
	}
}

func TestRemoveHelpMentionsCascade(t *testing.T) {
	t.Parallel()

	out, _, err := executeCommand(t, "remove", "--help")
	if err != nil {
		t.Fatalf("help: %v", err)
	}

	for _, want := range []string{"--cascade", "requires"} {
		if !strings.Contains(out, want) {
			t.Fatalf("remove help missing %q:\n%s", want, out)
		}
	}
}

func TestCompletionCommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		shell string
		want  string
	}{
		{shell: "bash", want: "_smokeforge"},
		{shell: "zsh", want: "#compdef smokeforge"},
		{shell: "fish", want: "complete -c smokeforge"},
		{shell: "powershell", want: "Register-ArgumentCompleter"},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			t.Parallel()

			out, _, err := executeCommand(t, "completion", tt.shell)
			if err != nil {
				t.Fatalf("completion %s: %v", tt.shell, err)
			}

			if !strings.Contains(out, tt.want) {
				t.Fatalf("completion output missing %q:\n%s", tt.want, out)
			}
		})
	}
}
