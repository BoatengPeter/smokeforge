# smokeforge Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A working `smokeforge` CLI that loads a versioned template-registry manifest from a local directory or remote tarball, validates it strictly, and exposes `version`, `list`, `registry`, and `doctor` commands.

**Architecture:** Cobra command layer on top of two libraries: `internal/manifest` (strict YAML schema + validation) and `internal/registry` (local/HTTP sources, checksum verification, safe extraction, XDG cache). All production code is TDD-driven.

**Tech Stack:** Go 1.25, Cobra, `gopkg.in/yaml.v3`, standard library HTTP/tar/gzip, `httptest` for tests, golangci-lint v2.

---

## File Structure

```
cmd/smokeforge/main.go          entrypoint
internal/version/version.go     build version vars
internal/cli/root.go            root command + persistent registry flags
internal/cli/version.go         version command
internal/cli/list.go            list command
internal/cli/registry.go        registry info/update
internal/cli/doctor.go          environment checks
internal/manifest/manifest.go   schema types + Load
internal/manifest/validate.go   strict validation
internal/registry/source.go     Source interface + Load dispatch
internal/registry/local.go      local directory source
internal/registry/http.go       remote tarball source + checksum
internal/registry/extract.go    safe tar.gz extraction
internal/registry/cache.go      cache dir resolution
.github/workflows/ci.yml        lint + test
Makefile                        build/test/lint targets
```

---

### Task 1: Repository scaffolding

**Files:**
- Create: `.gitignore`, `README.md`, `Makefile`, `.github/workflows/ci.yml`
- Existing: `go.mod`, `docs/superpowers/specs/2026-11-12-smokeforge-design.md`

- [ ] **Step 1: Write `.gitignore`**

```gitignore
/bin/
/dist/
*.test
*.out
.smokeforge/
```

- [ ] **Step 2: Write `Makefile`**

```makefile
.PHONY: build test lint fmt

build:
	go build -o bin/smokeforge ./cmd/smokeforge

test:
	go test ./...

lint:
	golangci-lint run

fmt:
	gofmt -w .
```

- [ ] **Step 3: Write `README.md`**

```markdown
# smokeforge

Scaffold backend, frontend, mobile, desktop, and fullstack projects from a
versioned template registry with composable feature overlays.

## Status

Foundation: registry manifest loading, validation, caching, and inspection
commands. Project generation lands in the next milestone.

## Build

    make build
    ./bin/smokeforge version
    ./bin/smokeforge list stacks --registry ./testdata/registry

## Development

    make test
    make lint
```

- [ ] **Step 4: Write `.github/workflows/ci.yml`**

```yaml
name: ci
on:
  push:
    branches: [main]
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - run: go test ./...
      - uses: golangci/golangci-lint-action@v7
        with:
          version: v2.11.0
```

- [ ] **Step 5: Commit**

```bash
git add .gitignore README.md Makefile .github docs go.mod
git commit -m "chore: scaffold smokeforge repository"
```

---

### Task 2: Version package and CLI root

**Files:**
- Create: `cmd/smokeforge/main.go`, `internal/version/version.go`
- Create: `internal/cli/root.go`, `internal/cli/version.go`
- Test: `internal/cli/version_test.go`

- [ ] **Step 1: Write the failing test**

`internal/cli/version_test.go`:

```go
package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	var out, errOut bytes.Buffer
	cmd := NewRootCmd(&out, &errOut)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "smokeforge") {
		t.Fatalf("version output missing program name: %q", got)
	}
	if !strings.Contains(got, versionForTest()) {
		t.Fatalf("version output missing version %q: %q", versionForTest(), got)
	}
}
```

Add a small test hook in the same file:

```go
func versionForTest() string { return "test-version" }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run TestVersionCommand -v`
Expected: FAIL / build error because `NewRootCmd` is undefined.

- [ ] **Step 3: Implement version package**

`internal/version/version.go`:

```go
package version

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func String() string {
	return Version + " (" + Commit + ", " + Date + ")"
}
```

- [ ] **Step 4: Implement root and version commands**

`internal/cli/root.go`:

```go
package cli

import (
	"io"

	"github.com/spf13/cobra"
)

type options struct {
	out      io.Writer
	errOut   io.Writer
	registry string
	tag      string
	offline  bool
}

func NewRootCmd(out, errOut io.Writer) *cobra.Command {
	opts := &options{out: out, errOut: errOut}

	root := &cobra.Command{
		Use:           "smokeforge",
		Short:         "Scaffold projects from a versioned template registry",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&opts.registry, "registry", "", "registry URL or local directory")
	root.PersistentFlags().StringVar(&opts.tag, "tag", "latest", "registry version tag")
	root.PersistentFlags().BoolVar(&opts.offline, "offline", false, "use cached registry only")

	root.AddCommand(newVersionCmd(out), newListCmd(opts), newRegistryCmd(opts), newDoctorCmd(opts))

	return root
}
```

`internal/cli/version.go`:

```go
package cli

import (
	"fmt"
	"io"

	"github.com/BoatengPeter/smokeforge/internal/version"
	"github.com/spf13/cobra"
)

func newVersionCmd(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(out, "smokeforge %s\n", version.String())
			return err
		},
	}
}
```

For the test to see `test-version`, set `version.Version` inside the test:

```go
func TestVersionCommand(t *testing.T) {
	old := version.Version
	version.Version = "test-version"
	t.Cleanup(func() { version.Version = old })
	// ... rest of test unchanged
}
```

`cmd/smokeforge/main.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/BoatengPeter/smokeforge/internal/cli"
)

func main() {
	if err := cli.NewRootCmd(os.Stdout, os.Stderr).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
```

Temporary stubs to keep the package compiling (replaced in later tasks):

```go
// internal/cli/list.go
package cli

import (
	"github.com/spf13/cobra"
)

func newListCmd(opts *options) *cobra.Command {
	return &cobra.Command{Use: "list", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
}
```

```go
// internal/cli/registry.go
package cli

import "github.com/spf13/cobra"

func newRegistryCmd(opts *options) *cobra.Command {
	return &cobra.Command{Use: "registry", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
}
```

```go
// internal/cli/doctor.go
package cli

import "github.com/spf13/cobra"

func newDoctorCmd(opts *options) *cobra.Command {
	return &cobra.Command{Use: "doctor", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/cli/ -run TestVersionCommand -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add cmd internal go.mod go.sum
git commit -m "feat: add CLI entrypoint and version command"
```

---

### Task 3: Manifest schema and loader

**Files:**
- Create: `internal/manifest/manifest.go`
- Test: `internal/manifest/manifest_test.go`
- Test data: `internal/manifest/testdata/manifest.yaml`

- [ ] **Step 1: Write the failing test**

`internal/manifest/manifest_test.go`:

```go
package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidManifest(t *testing.T) {
	m, err := LoadFile(filepath.Join("testdata", "manifest.yaml"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if m.SchemaVersion != 1 {
		t.Fatalf("schemaVersion = %d, want 1", m.SchemaVersion)
	}
	if len(m.Stacks) != 2 || m.Stacks[0].ID != "go/chi" {
		t.Fatalf("unexpected stacks: %+v", m.Stacks)
	}
	if len(m.Features) != 1 || m.Features[0].ID != "database" {
		t.Fatalf("unexpected features: %+v", m.Features)
	}
}

func TestLoadFileRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.yaml")
	data := []byte("schemaVersion: 1\nname: x\nunknown: true\n")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFile(path); err == nil {
		t.Fatal("expected unknown-field error")
	}
}
```

`internal/manifest/testdata/manifest.yaml`:

```yaml
schemaVersion: 1
name: test-registry
kinds:
  backend:
    max: 1
stacks:
  - id: go/chi
    kind: backend
    name: Go (chi)
    template: templates/backend/go/chi
    variables: [module_path, port]
  - id: rust/axum
    kind: backend
    name: Rust (axum)
    template: templates/backend/rust/axum
    variables: [crate_name, port]
features:
  - id: database
    name: Database
    kinds: [backend]
    stacks: [go/chi, rust/axum]
    variants:
      - id: postgres
        name: PostgreSQL
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/manifest/ -run TestLoad -v`
Expected: FAIL / build error because package `manifest` does not exist.

- [ ] **Step 3: Implement schema and loader**

`internal/manifest/manifest.go`:

```go
package manifest

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Manifest struct {
	SchemaVersion int                `yaml:"schemaVersion"`
	Name          string             `yaml:"name"`
	Kinds         map[string]Kind    `yaml:"kinds"`
	Stacks        []Stack            `yaml:"stacks"`
	Features      []Feature          `yaml:"features"`
}

type Kind struct {
	Max int `yaml:"max"`
}

type Stack struct {
	ID        string   `yaml:"id"`
	Kind      string   `yaml:"kind"`
	Name      string   `yaml:"name"`
	Template  string   `yaml:"template"`
	Variables []string `yaml:"variables"`
}

type Feature struct {
	ID        string    `yaml:"id"`
	Name      string    `yaml:"name"`
	Kinds     []string  `yaml:"kinds"`
	Stacks    []string  `yaml:"stacks"`
	Variants  []Variant `yaml:"variants"`
	Requires  []string  `yaml:"requires"`
	Conflicts []string  `yaml:"conflicts"`
}

type Variant struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

func Parse(data []byte) (*Manifest, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	var m Manifest
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &m, nil
}

func LoadFile(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	return Parse(data)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/manifest/ -run TestLoad -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/manifest
git commit -m "feat: add registry manifest schema and loader"
```

---

### Task 4: Manifest validation

**Files:**
- Create: `internal/manifest/validate.go`
- Test: `internal/manifest/validate_test.go`

- [ ] **Step 1: Write the failing tests**

`internal/manifest/validate_test.go`:

```go
package manifest

import (
	"strings"
	"testing"
)

func validManifest() *Manifest {
	return &Manifest{
		SchemaVersion: 1,
		Name:          "test",
		Kinds:         map[string]Kind{"backend": {Max: 1}},
		Stacks: []Stack{{
			ID: "go/chi", Kind: "backend", Name: "Go", Template: "templates/go", Variables: []string{"module_path"},
		}},
		Features: []Feature{{
			ID: "database", Name: "Database", Kinds: []string{"backend"},
			Variants: []Variant{{ID: "postgres", Name: "PostgreSQL"}},
		}},
	}
}

func TestValidateAcceptsValidManifest(t *testing.T) {
	if err := validManifest().Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestValidateRejects(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Manifest)
		want   string
	}{
		{"schema version", func(m *Manifest) { m.SchemaVersion = 2 }, "schemaVersion"},
		{"empty name", func(m *Manifest) { m.Name = "" }, "name"},
		{"unknown kind", func(m *Manifest) { m.Stacks[0].Kind = "quantum" }, "kind"},
		{"duplicate stack", func(m *Manifest) { m.Stacks = append(m.Stacks, m.Stacks[0]) }, "duplicate stack"},
		{"empty template", func(m *Manifest) { m.Stacks[0].Template = "" }, "template"},
		{"duplicate feature", func(m *Manifest) { m.Features = append(m.Features, m.Features[0]) }, "duplicate feature"},
		{"unknown requirement", func(m *Manifest) { m.Features[0].Requires = []string{"auth:jwt"} }, "unknown feature"},
		{"unknown stack reference", func(m *Manifest) { m.Features[0].Stacks = []string{"nope"} }, "unknown stack"},
		{"duplicate variant", func(m *Manifest) { m.Features[0].Variants = append(m.Features[0].Variants, m.Features[0].Variants[0]) }, "duplicate variant"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := validManifest()
			tt.mutate(m)
			err := m.Validate()
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error %q does not contain %q", err, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/manifest/ -run TestValidate -v`
Expected: FAIL / build error because `Validate` is undefined.

- [ ] **Step 3: Implement validation**

`internal/manifest/validate.go`:

```go
package manifest

import (
	"fmt"
	"strings"
)

func (m *Manifest) Validate() error {
	if m.SchemaVersion != 1 {
		return fmt.Errorf("unsupported schemaVersion %d", m.SchemaVersion)
	}
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("name must not be empty")
	}
	if len(m.Kinds) == 0 {
		return fmt.Errorf("kinds must not be empty")
	}
	if len(m.Stacks) == 0 {
		return fmt.Errorf("stacks must not be empty")
	}

	kindNames := make(map[string]bool, len(m.Kinds))
	for name, kind := range m.Kinds {
		if name == "" {
			return fmt.Errorf("kind name must not be empty")
		}
		if kind.Max < 0 {
			return fmt.Errorf("kind %q max must be >= 0", name)
		}
		kindNames[name] = true
	}

	stackIDs := make(map[string]bool, len(m.Stacks))
	for i, s := range m.Stacks {
		if s.ID == "" {
			return fmt.Errorf("stack %d: id must not be empty", i)
		}
		if stackIDs[s.ID] {
			return fmt.Errorf("duplicate stack %q", s.ID)
		}
		if !kindNames[s.Kind] {
			return fmt.Errorf("stack %q: unknown kind %q", s.ID, s.Kind)
		}
		if s.Template == "" {
			return fmt.Errorf("stack %q: template must not be empty", s.ID)
		}
		if len(s.Variables) == 0 {
			return fmt.Errorf("stack %q: variables must not be empty", s.ID)
		}
		seenVar := map[string]bool{}
		for _, v := range s.Variables {
			if v == "" || seenVar[v] {
				return fmt.Errorf("stack %q: invalid variable %q", s.ID, v)
			}
			seenVar[v] = true
		}
		stackIDs[s.ID] = true
	}

	featureIDs := make(map[string]bool, len(m.Features))
	for i, f := range m.Features {
		if f.ID == "" {
			return fmt.Errorf("feature %d: id must not be empty", i)
		}
		if featureIDs[f.ID] {
			return fmt.Errorf("duplicate feature %q", f.ID)
		}
		if len(f.Kinds) == 0 && len(f.Stacks) == 0 {
			return fmt.Errorf("feature %q: kinds or stacks required", f.ID)
		}
		for _, k := range f.Kinds {
			if !kindNames[k] {
				return fmt.Errorf("feature %q: unknown kind %q", f.ID, k)
			}
		}
		for _, s := range f.Stacks {
			if !stackIDs[s] {
				return fmt.Errorf("feature %q: unknown stack %q", f.ID, s)
			}
		}
		seenVariant := map[string]bool{}
		for _, v := range f.Variants {
			if v.ID == "" || seenVariant[v.ID] {
				return fmt.Errorf("feature %q: invalid or duplicate variant %q", f.ID, v.ID)
			}
			seenVariant[v.ID] = true
		}
		featureIDs[f.ID] = true
	}

	for _, f := range m.Features {
		for _, ref := range append(append([]string{}, f.Requires...), f.Conflicts...) {
			id := strings.SplitN(ref, ":", 2)[0]
			if id == f.ID {
				return fmt.Errorf("feature %q: cannot reference itself", f.ID)
			}
			if !featureIDs[id] {
				return fmt.Errorf("feature %q: unknown feature %q", f.ID, id)
			}
		}
	}

	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/manifest/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/manifest
git commit -m "feat: validate registry manifests"
```

---

### Task 5: Registry local source and loader dispatch

**Files:**
- Create: `internal/registry/source.go`, `internal/registry/local.go`
- Test: `internal/registry/local_test.go`

- [ ] **Step 1: Write the failing test**

`internal/registry/local_test.go`:

```go
package registry

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLocalDir(t *testing.T) {
	dir := t.TempDir()
	data := `schemaVersion: 1
name: local
kinds:
  backend: {max: 1}
stacks:
  - {id: "go/chi", kind: backend, name: Go, template: templates/go, variables: [module_path]}
features: []
`
	if err := os.WriteFile(filepath.Join(dir, "manifest.yaml"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Load(context.Background(), Options{Source: dir})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if res.Manifest.Name != "local" {
		t.Fatalf("name = %q", res.Manifest.Name)
	}
	if res.Tag != "local" {
		t.Fatalf("tag = %q, want local", res.Tag)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/registry/ -run TestLoadLocalDir -v`
Expected: FAIL / build error because `Load` and `Options` are undefined.

- [ ] **Step 3: Implement source dispatch and local source**

`internal/registry/source.go`:

```go
package registry

import (
	"context"
	"fmt"
	"os"

	"github.com/BoatengPeter/smokeforge/internal/manifest"
)

const DefaultURL = "https://github.com/BoatengPeter/smokeforge-registry/releases"

type Options struct {
	Source   string
	Tag      string
	CacheDir string
	Offline  bool
}

type Result struct {
	Manifest *manifest.Manifest
	Source   string
	Tag      string
	Dir      string
	FromCache bool
}

func Load(ctx context.Context, opts Options) (*Result, error) {
	if opts.Source == "" {
		opts.Source = DefaultURL
	}
	if fi, err := os.Stat(opts.Source); err == nil && fi.IsDir() {
		return loadLocal(opts.Source)
	}
	return loadRemote(ctx, opts)
}

func loadLocal(dir string) (*Result, error) {
	m, err := manifest.LoadFile(dir + "/manifest.yaml")
	if err != nil {
		return nil, fmt.Errorf("load local registry: %w", err)
	}
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("validate local registry: %w", err)
	}
	return &Result{Manifest: m, Source: dir, Tag: "local", Dir: dir}, nil
}
```

`internal/registry/local.go` can be omitted; keep loadLocal in source.go for now. Create the file with the package clause only? Avoid empty file. Skip local.go; update file structure accordingly.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/registry/ -run TestLoadLocalDir -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/registry
git commit -m "feat: load registries from local directories"
```

---

### Task 6: Remote tarball source with checksum and cache

**Files:**
- Create: `internal/registry/http.go`, `internal/registry/extract.go`, `internal/registry/cache.go`
- Test: `internal/registry/http_test.go`, `internal/registry/extract_test.go`

- [ ] **Step 1: Write the failing extraction tests**

`internal/registry/extract_test.go`:

```go
package registry

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func tarGz(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range entries {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(content))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestExtractWritesFiles(t *testing.T) {
	dst := t.TempDir()
	data := tarGz(t, map[string]string{"manifest.yaml": "schemaVersion: 1\n"})
	if err := Extract(bytes.NewReader(data), dst); err != nil {
		t.Fatalf("extract: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dst, "manifest.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "schemaVersion: 1\n" {
		t.Fatalf("content = %q", got)
	}
}

func TestExtractRejectsPathTraversal(t *testing.T) {
	dst := t.TempDir()
	data := tarGz(t, map[string]string{"../escape": "x"})
	if err := Extract(bytes.NewReader(data), dst); err == nil {
		t.Fatal("expected path traversal error")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/registry/ -run TestExtract -v`
Expected: FAIL / build error because `Extract` is undefined.

- [ ] **Step 3: Implement safe extraction**

`internal/registry/extract.go`:

```go
package registry

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func Extract(r io.Reader, dst string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("open gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}

		name := filepath.Clean(hdr.Name)
		if filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
			return fmt.Errorf("unsafe path %q", hdr.Name)
		}
		if hdr.Typeflag == tar.TypeSymlink || hdr.Typeflag == tar.TypeLink {
			return fmt.Errorf("links are not allowed: %q", hdr.Name)
		}
		target := filepath.Join(dst, name)
		if hdr.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if hdr.Typeflag != tar.TypeReg {
			return fmt.Errorf("unsupported entry %q", hdr.Name)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)&0o644)
		if err != nil {
			return err
		}
		if _, err := io.Copy(f, tr); err != nil {
			f.Close()
			return fmt.Errorf("write %q: %w", hdr.Name, err)
		}
		if err := f.Close(); err != nil {
			return err
		}
	}
}
```

- [ ] **Step 4: Run extraction tests**

Run: `go test ./internal/registry/ -run TestExtract -v`
Expected: PASS

- [ ] **Step 5: Write the failing HTTP source tests**

`internal/registry/http_test.go`:

```go
package registry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const remoteManifest = `schemaVersion: 1
name: remote
kinds:
  backend: {max: 1}
stacks:
  - {id: "go/chi", kind: backend, name: Go, template: templates/go, variables: [module_path]}
features: []
`

func remoteServer(t *testing.T) *httptest.Server {
	t.Helper()
	data := tarGz(t, map[string]string{"registry/manifest.yaml": remoteManifest})
	sum := sha256.Sum256(data)

	mux := http.NewServeMux()
	mux.HandleFunc("/releases/download/v1.0.0/registry.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	})
	mux.HandleFunc("/releases/download/v1.0.0/registry.sha256", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(hex.EncodeToString(sum[:]) + "  registry.tar.gz\n"))
	})
	return httptest.NewServer(mux)
}

func TestLoadRemoteVerifiesAndCaches(t *testing.T) {
	srv := remoteServer(t)
	defer srv.Close()

	cache := t.TempDir()
	opts := Options{Source: srv.URL + "/releases", Tag: "v1.0.0", CacheDir: cache}

	res, err := Load(context.Background(), opts)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if res.Manifest.Name != "remote" {
		t.Fatalf("name = %q", res.Manifest.Name)
	}

	// Second load with the server stopped and offline set must use cache.
	srv.Close()
	opts.Offline = true
	res, err = Load(context.Background(), opts)
	if err != nil {
		t.Fatalf("offline load: %v", err)
	}
	if res.Manifest.Name != "remote" {
		t.Fatalf("offline name = %q", res.Manifest.Name)
	}
	if !res.FromCache {
		t.Fatal("expected FromCache to be true")
	}
}

func TestLoadRemoteRejectsBadChecksum(t *testing.T) {
	data := tarGz(t, map[string]string{"registry/manifest.yaml": remoteManifest})
	mux := http.NewServeMux()
	mux.HandleFunc("/releases/download/v1.0.0/registry.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	})
	mux.HandleFunc("/releases/download/v1.0.0/registry.sha256", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Repeat("0", 64) + "  registry.tar.gz\n"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	_, err := Load(context.Background(), Options{Source: srv.URL + "/releases", Tag: "v1.0.0", CacheDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("expected checksum error, got %v", err)
	}
}

func TestLoadRemoteOfflineWithoutCacheFails(t *testing.T) {
	_, err := Load(context.Background(), Options{Source: "http://127.0.0.1:1/releases", Tag: "v1.0.0", CacheDir: t.TempDir(), Offline: true})
	if err == nil || !strings.Contains(err.Error(), "offline") {
		t.Fatalf("expected offline error, got %v", err)
	}
}

func TestLoadRemoteRejectsUnsafeTag(t *testing.T) {
	_, err := Load(context.Background(), Options{Source: "http://127.0.0.1:1/releases", Tag: "../x", CacheDir: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "tag") {
		t.Fatalf("expected tag error, got %v", err)
	}
}

func TestLoadRemoteUsesCachedManifestWhenOnline(t *testing.T) {
	srv := remoteServer(t)
	defer srv.Close()
	cache := t.TempDir()
	opts := Options{Source: srv.URL + "/releases", Tag: "v1.0.0", CacheDir: cache}
	if _, err := Load(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	res, err := Load(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if !res.FromCache {
		t.Fatal("expected FromCache to be true")
	}
}

func TestDefaultCacheDirUnderUserCache(t *testing.T) {
	dir, err := DefaultCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(dir, "smokeforge") {
		t.Fatalf("cache dir = %q", dir)
	}
	_ = os.RemoveAll(filepath.Join(dir, "test-nonexistent"))
}
```

- [ ] **Step 6: Run tests to verify they fail**

Run: `go test ./internal/registry/ -run 'TestLoadRemote|TestDefaultCacheDir' -v`
Expected: FAIL because `loadRemote`, `DefaultCacheDir` are undefined.

- [ ] **Step 7: Implement remote loading and cache**

`internal/registry/cache.go`:

```go
package registry

import (
	"fmt"
	"os"
	"path/filepath"
)

func DefaultCacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve user cache dir: %w", err)
	}
	return filepath.Join(base, "smokeforge", "registry"), nil
}

func entryDir(cacheDir, source, tag string) string {
	name := fmt.Sprintf("%x", sha256Hex(source+"\x00"+tag))[:16]
	return filepath.Join(cacheDir, name)
}

func sha256Hex(s string) []byte {
	sum := sha256.Sum256([]byte(s))
	return sum[:]
}
```

Add `"crypto/sha256"` to imports.

`internal/registry/http.go`:

```go
package registry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/BoatengPeter/smokeforge/internal/manifest"
)

var tagPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func loadRemote(ctx context.Context, opts Options) (*Result, error) {
	if !tagPattern.MatchString(opts.Tag) {
		return nil, fmt.Errorf("invalid registry tag %q", opts.Tag)
	}
	cacheDir := opts.CacheDir
	if cacheDir == "" {
		var err error
		cacheDir, err = DefaultCacheDir()
		if err != nil {
			return nil, err
		}
	}
	dir := entryDir(cacheDir, opts.Source, opts.Tag)

	if res, err := loadCached(opts, dir); err == nil {
		return res, nil
	}
	if opts.Offline {
		return nil, fmt.Errorf("registry %s@%s not in cache and offline mode is enabled", opts.Source, opts.Tag)
	}
	if err := download(ctx, opts, dir); err != nil {
		return nil, err
	}
	return loadCached(opts, dir)
}

func loadCached(opts Options, dir string) (*Result, error) {
	m, err := manifest.LoadFile(filepath.Join(dir, "manifest.yaml"))
	if err != nil {
		return nil, err
	}
	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("validate registry: %w", err)
	}
	return &Result{Manifest: m, Source: opts.Source, Tag: opts.Tag, Dir: dir, FromCache: true}, nil
}

func download(ctx context.Context, opts Options, dir string) error {
	client := &http.Client{Timeout: 2 * time.Minute}

	base := strings.TrimSuffix(opts.Source, "/") + "/download/" + opts.Tag
	wantSum, err := fetchChecksum(ctx, client, base+"/registry.sha256")
	if err != nil {
		return err
	}
	body, err := fetch(ctx, client, base+"/registry.tar.gz")
	if err != nil {
		body.Close()
		return err
	}
	defer body.Close()

	tmp, err := os.MkdirTemp(filepath.Dir(dir), ".download-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmp)

	archive := filepath.Join(tmp, "registry.tar.gz")
	f, err := os.Create(archive)
	if err != nil {
		return err
	}
	hasher := sha256.New()
	if _, err := io.Copy(io.MultiWriter(f, hasher), body); err != nil {
		f.Close()
		return fmt.Errorf("download registry: %w", err)
	}
	if err := f.Close(); err != nil {
		return err
	}
	if got := hex.EncodeToString(hasher.Sum(nil)); got != wantSum {
		return fmt.Errorf("checksum mismatch: got %s want %s", got, wantSum)
	}

	extracted := filepath.Join(tmp, "registry")
	if err := os.MkdirAll(extracted, 0o755); err != nil {
		return err
	}
	af, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer af.Close()
	if err := Extract(af, extracted); err != nil {
		return err
	}
	if err := os.Rename(extracted, dir); err != nil {
		return fmt.Errorf("commit registry cache: %w", err)
	}
	return nil
}

func fetchChecksum(ctx context.Context, client *http.Client, url string) (string, error) {
	body, err := fetch(ctx, client, url)
	if err != nil {
		return "", err
	}
	defer body.Close()
	data, err := io.ReadAll(io.LimitReader(body, 4096))
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 || len(fields[0]) != 64 {
		return "", fmt.Errorf("invalid checksum file from %s", url)
	}
	return strings.ToLower(fields[0]), nil
}

func fetch(ctx context.Context, client *http.Client, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("fetch %s: status %s", url, resp.StatusCode)
	}
	return resp.Body, nil
}
```

Note: the test archive uses `registry/manifest.yaml`; `Extract` writes into `extracted/registry/manifest.yaml`. The cache lookup expects `dir/manifest.yaml`. Fix `download` to extract into `tmp` and then rename `tmp/registry` to `dir`, i.e.:

```go
if err := os.MkdirAll(extracted, 0o755); err != nil { return err }
...
if err := Extract(af, extracted); err != nil { return err }
if err := os.Rename(extracted, dir); err != nil { ... }
```

`extracted` is `tmp/registry`, so contents land at `dir/manifest.yaml`. Correct.

- [ ] **Step 8: Run registry tests**

Run: `go test ./internal/registry/ -v`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add internal/registry go.mod go.sum
git commit -m "feat: fetch, verify, and cache remote registries"
```

---

### Task 7: `smokeforge list`

**Files:**
- Modify: `internal/cli/list.go`
- Test: `internal/cli/list_test.go`

- [ ] **Step 1: Write the failing test**

```go
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestRegistry(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	data := `schemaVersion: 1
name: test
kinds:
  backend: {max: 1}
stacks:
  - {id: "go/chi", kind: backend, name: Go (chi), template: templates/go, variables: [module_path]}
features:
  - id: database
    name: Database
    kinds: [backend]
    variants:
      - {id: postgres, name: PostgreSQL}
`
	if err := os.WriteFile(filepath.Join(dir, "manifest.yaml"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestListStacks(t *testing.T) {
	var out, errOut bytes.Buffer
	cmd := NewRootCmd(&out, &errOut)
	cmd.SetArgs([]string{"list", "stacks", "--registry", writeTestRegistry(t)})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "go/chi") || !strings.Contains(got, "Go (chi)") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestListFeatures(t *testing.T) {
	var out, errOut bytes.Buffer
	cmd := NewRootCmd(&out, &errOut)
	cmd.SetArgs([]string{"list", "features", "--registry", writeTestRegistry(t)})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "database") || !strings.Contains(got, "postgres") {
		t.Fatalf("unexpected output: %q", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run TestList -v`
Expected: FAIL because the stub prints nothing.

- [ ] **Step 3: Implement `list`**

`internal/cli/list.go`:

```go
package cli

import (
	"context"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/BoatengPeter/smokeforge/internal/registry"
	"github.com/spf13/cobra"
)

func newListCmd(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "list [stacks|features]",
		Short: "List registry contents",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			what := "stacks"
			if len(args) == 1 {
				what = args[0]
			}
			if what != "stacks" && what != "features" {
				return fmt.Errorf("unknown list target %q (want stacks or features)", what)
			}

			res, err := registry.Load(cmd.Context(), registry.Options{
				Source:  opts.registry,
				Tag:     opts.tag,
				Offline: opts.offline,
			})
			if err != nil {
				return err
			}

			tw := tabwriter.NewWriter(opts.out, 0, 4, 2, ' ', 0)
			if what == "stacks" {
				fmt.Fprintln(tw, "ID\tKIND\tNAME")
				for _, s := range res.Manifest.Stacks {
					fmt.Fprintf(tw, "%s\t%s\t%s\n", s.ID, s.Kind, s.Name)
				}
			} else {
				fmt.Fprintln(tw, "ID\tVARIANTS\tREQUIRES")
				for _, f := range res.Manifest.Features {
					variants := make([]string, 0, len(f.Variants))
					for _, v := range f.Variants {
						variants = append(variants, v.ID)
					}
					fmt.Fprintf(tw, "%s\t%s\t%s\n", f.ID, strings.Join(variants, ","), strings.Join(f.Requires, ","))
				}
			}
			return tw.Flush()
		},
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/cli/ -run TestList -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli
git commit -m "feat: add list command"
```

---

### Task 8: `registry info`, `registry update`, and `doctor`

**Files:**
- Modify: `internal/cli/registry.go`, `internal/cli/doctor.go`
- Test: `internal/cli/registry_test.go`, `internal/cli/doctor_test.go`

- [ ] **Step 1: Write the failing tests**

`internal/cli/registry_test.go`:

```go
package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRegistryInfo(t *testing.T) {
	var out, errOut bytes.Buffer
	cmd := NewRootCmd(&out, &errOut)
	cmd.SetArgs([]string{"registry", "info", "--registry", writeTestRegistry(t)})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	got := out.String()
	for _, want := range []string{"source:", "tag: local", "schema: 1", "stacks: 1", "features: 1"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output %q missing %q", got, want)
		}
	}
}

func TestRegistryUnknownSubcommand(t *testing.T) {
	var out, errOut bytes.Buffer
	cmd := NewRootCmd(&out, &errOut)
	cmd.SetArgs([]string{"registry", "nope"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error")
	}
}
```

`internal/cli/doctor_test.go`:

```go
package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestDoctorReportsChecks(t *testing.T) {
	var out, errOut bytes.Buffer
	cmd := NewRootCmd(&out, &errOut)
	cmd.SetArgs([]string{"doctor", "--registry", writeTestRegistry(t)})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	got := out.String()
	for _, want := range []string{"registry", "git", "ok"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output %q missing %q", got, want)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run 'TestRegistry|TestDoctor' -v`
Expected: FAIL because the stubs print nothing.

- [ ] **Step 3: Implement `registry` and `doctor`**

`internal/cli/registry.go`:

```go
package cli

import (
	"fmt"

	"github.com/BoatengPeter/smokeforge/internal/registry"
	"github.com/spf13/cobra"
)

func newRegistryCmd(opts *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Inspect and update the template registry",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "info",
			Short: "Show registry information",
			RunE: func(cmd *cobra.Command, args []string) error {
				res, err := registry.Load(cmd.Context(), registry.Options{
					Source:  opts.registry,
					Tag:     opts.tag,
					Offline: opts.offline,
				})
				if err != nil {
					return err
				}
				fmt.Fprintf(opts.out, "source: %s\n", res.Source)
				fmt.Fprintf(opts.out, "tag: %s\n", res.Tag)
				fmt.Fprintf(opts.out, "schema: %d\n", res.Manifest.SchemaVersion)
				fmt.Fprintf(opts.out, "stacks: %d\n", len(res.Manifest.Stacks))
				fmt.Fprintf(opts.out, "features: %d\n", len(res.Manifest.Features))
				return nil
			},
		},
		&cobra.Command{
			Use:   "update",
			Short: "Refresh the cached registry",
			RunE: func(cmd *cobra.Command, args []string) error {
				if opts.offline {
					return fmt.Errorf("cannot update in offline mode")
				}
				cacheDir, err := registry.DefaultCacheDir()
				if err != nil {
					return err
				}
				res, err := registry.Refresh(cmd.Context(), registry.Options{
					Source:   opts.registry,
					Tag:      opts.tag,
					CacheDir: cacheDir,
				})
				if err != nil {
					return err
				}
				fmt.Fprintf(opts.out, "updated registry %s@%s\n", res.Source, res.Tag)
				return nil
			},
		},
	)
	return cmd
}
```

Add `Refresh` to `internal/registry/source.go`:

```go
func Refresh(ctx context.Context, opts Options) (*Result, error) {
	if opts.Source == "" {
		opts.Source = DefaultURL
	}
	if fi, err := os.Stat(opts.Source); err == nil && fi.IsDir() {
		return loadLocal(opts.Source)
	}
	if opts.CacheDir == "" {
		var err error
		opts.CacheDir, err = DefaultCacheDir()
		if err != nil {
			return nil, err
		}
	}
	dir := entryDir(opts.CacheDir, opts.Source, opts.Tag)
	if err := os.RemoveAll(dir); err != nil {
		return nil, fmt.Errorf("clear registry cache: %w", err)
	}
	if err := download(ctx, opts, dir); err != nil {
		return nil, err
	}
	return loadCached(opts, dir)
}
```

`internal/cli/doctor.go`:

```go
package cli

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/BoatengPeter/smokeforge/internal/registry"
	"github.com/spf13/cobra"
)

func newDoctorCmd(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check the local environment and registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			failed := false

			if _, err := registry.Load(cmd.Context(), registry.Options{
				Source:  opts.registry,
				Tag:     opts.tag,
				Offline: opts.offline,
			}); err != nil {
				fmt.Fprintf(opts.out, "registry: fail (%v)\n", err)
				failed = true
			} else {
				fmt.Fprintln(opts.out, "registry: ok")
			}

			if _, err := exec.LookPath("git"); err != nil {
				fmt.Fprintf(opts.out, "git: fail (%v)\n", err)
				failed = true
			} else {
				fmt.Fprintln(opts.out, "git: ok")
			}

			if failed {
				return fmt.Errorf("doctor found problems")
			}
			return nil
		},
	}
}
```

Remove the unused `context` import (use `cmd.Context()` only). Keep imports accurate.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/cli/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli internal/registry
git commit -m "feat: add registry inspection and doctor commands"
```

---

### Task 9: Full verification and CI

**Files:**
- Modify: `README.md` if command output differs
- Existing: `.github/workflows/ci.yml`

- [ ] **Step 1: Run the full test suite**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 2: Run lint**

Run: `golangci-lint run`
Expected: no issues

- [ ] **Step 3: Run the binary manually against the local fixture**

Run: `go run ./cmd/smokeforge list stacks --registry ./internal/manifest/testdata`
Expected: table with `go/chi` and `rust/axum`

- [ ] **Step 4: Commit any fixes**

```bash
git add -A
git commit -m "chore: finalize foundation milestone"
```
