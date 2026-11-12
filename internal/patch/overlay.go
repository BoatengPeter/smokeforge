// Package patch applies declarative feature overlays to a staged project tree.
package patch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/BoatengPeter/smokeforge/internal/render"
	"gopkg.in/yaml.v3"
)

// Overlay is a declarative set of operations that adds one feature variant.
type Overlay struct {
	ID    string `yaml:"id"`
	Name  string `yaml:"name"`
	Files string `yaml:"files"`
	Ops   []Op   `yaml:"ops"`
}

// Op holds exactly one overlay operation.
type Op struct {
	AddFiles      *AddFiles   `yaml:"addFiles"`
	InsertBefore  *Insert     `yaml:"insertBefore"`
	InsertAfter   *Insert     `yaml:"insertAfter"`
	PatchJSON     *JSONPatch  `yaml:"patchJSON"`
	PatchYAML     *YAMLPatch  `yaml:"patchYAML"`
	PatchTOML     *TOMLPatch  `yaml:"patchTOML"`
	AddDependency *Dependency `yaml:"addDependency"`
	Env           *Env        `yaml:"env"`
}

// AddFiles lists files copied from the overlay directory into the project.
type AddFiles []AddFile

// AddFile maps one overlay source file to a rendered destination path.
type AddFile struct {
	Path      string `yaml:"path"`
	Dest      string `yaml:"dest"`
	Overwrite bool   `yaml:"overwrite"`
}

// Insert adds a marked block before or after a named anchor.
type Insert struct {
	File    string `yaml:"file"`
	Anchor  string `yaml:"anchor"`
	Marker  string `yaml:"marker"`
	Content string `yaml:"content"`
}

// JSONPatch edits a JSON document at a dot-separated path.
type JSONPatch struct {
	File  string `yaml:"file"`
	Op    string `yaml:"op"`
	Path  string `yaml:"path"`
	Value any    `yaml:"value"`
}

// YAMLPatch edits a YAML document at a dot-separated path.
type YAMLPatch struct {
	File  string `yaml:"file"`
	Op    string `yaml:"op"`
	Path  string `yaml:"path"`
	Value any    `yaml:"value"`
}

// TOMLPatch edits a TOML document at a dot-separated path.
type TOMLPatch struct {
	File  string `yaml:"file"`
	Op    string `yaml:"op"`
	Path  string `yaml:"path"`
	Value any    `yaml:"value"`
}

// Dependency adds a module to a language dependency manifest.
type Dependency struct {
	File     string   `yaml:"file"`
	Module   string   `yaml:"module"`
	Version  string   `yaml:"version"`
	Features []string `yaml:"features"`
}

// Env documents an environment variable consumed by the generated project.
type Env struct {
	Key     string `yaml:"key"`
	Example string `yaml:"example"`
	Secret  bool   `yaml:"secret"`
}

// LoadOverlayFile reads and validates an overlay manifest.
func LoadOverlayFile(pathname string) (*Overlay, error) {
	//nolint:gosec // pathname is resolved from a validated registry overlay directory.
	data, err := os.ReadFile(pathname)
	if err != nil {
		return nil, fmt.Errorf("read overlay %q: %w", pathname, err)
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	var overlay Overlay
	if err := dec.Decode(&overlay); err != nil {
		return nil, fmt.Errorf("parse overlay %q: %w", pathname, err)
	}

	var extra any

	err = dec.Decode(&extra)
	switch {
	case err == nil:
		return nil, fmt.Errorf("parse overlay %q: multiple YAML documents are not supported", pathname)
	case errors.Is(err, io.EOF):
		return &overlay, nil
	default:
		return nil, fmt.Errorf("parse overlay %q: %w", pathname, err)
	}
}

// Validate checks identifiers, paths, and operation-specific requirements.
func (o *Overlay) Validate() error {
	if strings.TrimSpace(o.ID) == "" {
		return errors.New("overlay id must not be empty")
	}

	if len(o.Ops) == 0 {
		return fmt.Errorf("overlay %q: ops must not be empty", o.ID)
	}

	if o.Files != "" {
		if _, err := safeRelativePath(o.Files); err != nil {
			return fmt.Errorf("overlay %q: files: %w", o.ID, err)
		}
	}

	for index, op := range o.Ops {
		if err := op.validate(o.ID); err != nil {
			return fmt.Errorf("overlay %q: op %d: %w", o.ID, index, err)
		}
	}

	return nil
}

func (o Op) validate(overlayID string) error {
	set := 0

	for _, present := range []bool{
		o.AddFiles != nil,
		o.InsertBefore != nil,
		o.InsertAfter != nil,
		o.PatchJSON != nil,
		o.PatchYAML != nil,
		o.PatchTOML != nil,
		o.AddDependency != nil,
		o.Env != nil,
	} {
		if present {
			set++
		}
	}

	if set != 1 {
		return fmt.Errorf("overlay %q: each op must set exactly one operation", overlayID)
	}

	switch {
	case o.AddFiles != nil:
		return o.AddFiles.validate()
	case o.InsertBefore != nil:
		return o.InsertBefore.validate()
	case o.InsertAfter != nil:
		return o.InsertAfter.validate()
	case o.PatchJSON != nil:
		return o.PatchJSON.validate()
	case o.PatchYAML != nil:
		return o.PatchYAML.validate()
	case o.PatchTOML != nil:
		return o.PatchTOML.validate()
	case o.AddDependency != nil:
		return o.AddDependency.validate()
	case o.Env != nil:
		return o.Env.validate()
	default:
		return errors.New("no operation selected")
	}
}

func (a AddFiles) validate() error {
	if len(a) == 0 {
		return errors.New("addFiles must not be empty")
	}

	for _, file := range a {
		if _, err := safeRelativePath(file.Path); err != nil {
			return fmt.Errorf("addFiles source: %w", err)
		}

		if _, err := safeRelativePath(file.Dest); err != nil {
			return fmt.Errorf("addFiles destination: %w", err)
		}
	}

	return nil
}

func (i *Insert) validate() error {
	if _, err := safeRelativePath(i.File); err != nil {
		return fmt.Errorf("insert file: %w", err)
	}

	if strings.TrimSpace(i.Anchor) == "" {
		return errors.New("insert anchor must not be empty")
	}

	if i.Marker != "" {
		if strings.ContainsAny(i.Marker, " \t\n") {
			return fmt.Errorf("insert marker %q must not contain whitespace", i.Marker)
		}
	}

	return nil
}

func (p *JSONPatch) validate() error {
	return validateDocumentPatch(p.File, p.Op, p.Path)
}

func (p *YAMLPatch) validate() error {
	return validateDocumentPatch(p.File, p.Op, p.Path)
}

func (p *TOMLPatch) validate() error {
	return validateDocumentPatch(p.File, p.Op, p.Path)
}

func validateDocumentPatch(file, operation, pathValue string) error {
	if _, err := safeRelativePath(file); err != nil {
		return fmt.Errorf("patch file: %w", err)
	}

	switch operation {
	case "set", "merge", "append", "delete":
	default:
		return fmt.Errorf("patch operation %q must be set, merge, append, or delete", operation)
	}

	if strings.TrimSpace(pathValue) == "" {
		return errors.New("patch path must not be empty")
	}

	return nil
}

func (d *Dependency) validate() error {
	if _, err := safeRelativePath(d.File); err != nil {
		return fmt.Errorf("dependency file: %w", err)
	}

	if strings.TrimSpace(d.Module) == "" {
		return errors.New("dependency module must not be empty")
	}

	if strings.TrimSpace(d.Version) == "" {
		return errors.New("dependency version must not be empty")
	}

	return nil
}

func (e *Env) validate() error {
	if strings.TrimSpace(e.Key) == "" {
		return errors.New("env key must not be empty")
	}

	if strings.ContainsAny(e.Key, " \t\n=") {
		return fmt.Errorf("env key %q contains invalid characters", e.Key)
	}

	return nil
}

// ApplyOptions configures one overlay application.
type ApplyOptions struct {
	Root       string
	OverlayDir string
	Renderer   *render.Renderer
}

// Applied records the observable effects of an overlay for project state.
type Applied struct {
	Files   []string
	Markers []string
	Env     []Env
}

// Apply executes an overlay against a staged project tree. Paths are
// constrained to the project root, inserted blocks are marked, and every
// operation is verified as it is applied.
func Apply(ctx context.Context, overlay *Overlay, opts ApplyOptions) (*Applied, error) {
	if err := overlay.Validate(); err != nil {
		return nil, err
	}

	if opts.Renderer == nil {
		return nil, errors.New("apply overlay: renderer is required")
	}

	root, err := os.OpenRoot(opts.Root)
	if err != nil {
		return nil, fmt.Errorf("open project root: %w", err)
	}

	defer func() {
		_ = root.Close()
	}()

	applied := &Applied{
		Files:   make([]string, 0, len(overlay.Ops)),
		Markers: make([]string, 0, len(overlay.Ops)),
		Env:     make([]Env, 0, len(overlay.Ops)),
	}

	for index, op := range overlay.Ops {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("apply overlay %q: op %d: %w", overlay.ID, index, err)
		}

		if err := applyOp(root, overlay, opts, op, applied); err != nil {
			return nil, fmt.Errorf("apply overlay %q: op %d: %w", overlay.ID, index, err)
		}
	}

	return applied, nil
}

func applyOp(
	root *os.Root,
	overlay *Overlay,
	opts ApplyOptions,
	op Op,
	applied *Applied,
) error {
	switch {
	case op.AddFiles != nil:
		return applyAddFiles(root, overlay, opts, *op.AddFiles, applied)
	case op.InsertBefore != nil:
		return applyInsert(root, overlay, opts, op.InsertBefore, true, applied)
	case op.InsertAfter != nil:
		return applyInsert(root, overlay, opts, op.InsertAfter, false, applied)
	case op.PatchJSON != nil:
		return applyDocumentPatch(root, opts.Renderer, op.PatchJSON.File, func(data []byte) ([]byte, error) {
			return patchJSON(data, op.PatchJSON.Op, op.PatchJSON.Path, op.PatchJSON.Value)
		}, applied)
	case op.PatchYAML != nil:
		return applyDocumentPatch(root, opts.Renderer, op.PatchYAML.File, func(data []byte) ([]byte, error) {
			return patchYAML(data, op.PatchYAML.Op, op.PatchYAML.Path, op.PatchYAML.Value)
		}, applied)
	case op.PatchTOML != nil:
		return applyDocumentPatch(root, opts.Renderer, op.PatchTOML.File, func(data []byte) ([]byte, error) {
			return patchTOML(data, op.PatchTOML.Op, op.PatchTOML.Path, op.PatchTOML.Value)
		}, applied)
	case op.AddDependency != nil:
		return applyDependency(root, opts.Renderer, op.AddDependency, applied)
	case op.Env != nil:
		applied.Env = append(applied.Env, *op.Env)

		return nil
	default:
		return errors.New("no operation selected")
	}
}

func applyAddFiles(
	root *os.Root,
	overlay *Overlay,
	opts ApplyOptions,
	files AddFiles,
	applied *Applied,
) error {
	filesDir := overlay.Files
	if filesDir == "" {
		filesDir = "files"
	}

	filesDir = filepath.FromSlash(filesDir)

	for _, file := range files {
		source, err := safeJoin(opts.OverlayDir, filesDir, file.Path)
		if err != nil {
			return err
		}

		//nolint:gosec // source is constrained to the overlay directory by safeJoin.
		data, err := os.ReadFile(source)
		if err != nil {
			return fmt.Errorf("read overlay file %q: %w", file.Path, err)
		}

		if strings.HasSuffix(file.Path, ".tmpl") {
			data, err = opts.Renderer.RenderBytes(data)
			if err != nil {
				return fmt.Errorf("render overlay file %q: %w", file.Path, err)
			}
		}

		dest, err := opts.Renderer.RenderPath(file.Dest)
		if err != nil {
			return fmt.Errorf("render overlay destination %q: %w", file.Dest, err)
		}

		dest, err = safeRelativePath(dest)
		if err != nil {
			return err
		}

		if err := writeNewFile(root, dest, data, source, file.Overwrite); err != nil {
			return err
		}

		applied.Files = appendUnique(applied.Files, dest)
	}

	return nil
}

func writeNewFile(root *os.Root, dest string, data []byte, source string, overwrite bool) error {
	mode := os.FileMode(0o644)

	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("stat overlay file %q: %w", source, err)
	}

	if info.Mode()&0o111 != 0 {
		mode = 0o755
	}

	_, statErr := root.Stat(filepath.FromSlash(dest))

	exists := statErr == nil
	if exists && !overwrite {
		return fmt.Errorf("destination %q already exists", dest)
	}

	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("stat destination %q: %w", dest, statErr)
	}

	if dir := path.Dir(dest); dir != "." {
		if err := root.MkdirAll(filepath.FromSlash(dir), 0o755); err != nil {
			return fmt.Errorf("create parent directory for %q: %w", dest, err)
		}
	}

	return writeFile(root, dest, data, mode)
}

func applyInsert(
	root *os.Root,
	overlay *Overlay,
	opts ApplyOptions,
	insert *Insert,
	before bool,
	applied *Applied,
) error {
	target, err := opts.Renderer.RenderPath(insert.File)
	if err != nil {
		return fmt.Errorf("render insert file %q: %w", insert.File, err)
	}

	data, mode, err := readFile(root, target)
	if err != nil {
		return err
	}

	block, err := opts.Renderer.RenderBytes([]byte(insert.Content))
	if err != nil {
		return fmt.Errorf("render insert content for %q: %w", target, err)
	}

	marker := insert.Marker
	if marker == "" {
		marker = overlay.ID
	}

	updated, err := insertBlock(insertRequest{
		file:    target,
		content: string(data),
		anchor:  insert.Anchor,
		marker:  marker,
		block:   string(block),
		before:  before,
	})
	if err != nil {
		return err
	}

	if err := writeFile(root, target, []byte(updated), mode); err != nil {
		return err
	}

	applied.Files = appendUnique(applied.Files, target)
	applied.Markers = appendUnique(applied.Markers, marker)

	return nil
}

func applyDocumentPatch(
	root *os.Root,
	renderer *render.Renderer,
	file string,
	patch func([]byte) ([]byte, error),
	applied *Applied,
) error {
	target, err := renderer.RenderPath(file)
	if err != nil {
		return fmt.Errorf("render patch file %q: %w", file, err)
	}

	data, mode, err := readFile(root, target)
	if err != nil {
		return err
	}

	updated, err := patch(data)
	if err != nil {
		return fmt.Errorf("patch %q: %w", target, err)
	}

	if err := writeFile(root, target, updated, mode); err != nil {
		return err
	}

	applied.Files = appendUnique(applied.Files, target)

	return nil
}

func applyDependency(
	root *os.Root,
	renderer *render.Renderer,
	dependency *Dependency,
	applied *Applied,
) error {
	target, err := renderer.RenderPath(dependency.File)
	if err != nil {
		return fmt.Errorf("render dependency file %q: %w", dependency.File, err)
	}

	data, mode, err := readFile(root, target)
	if err != nil {
		return err
	}

	updated, err := addDependency(
		data,
		target,
		dependency.Module,
		dependency.Version,
		dependency.Features,
	)
	if err != nil {
		return err
	}

	if err := writeFile(root, target, updated, mode); err != nil {
		return err
	}

	applied.Files = appendUnique(applied.Files, target)

	return nil
}

func readFile(root *os.Root, target string) ([]byte, os.FileMode, error) {
	info, err := root.Stat(filepath.FromSlash(target))
	if err != nil {
		return nil, 0, fmt.Errorf("stat file %q: %w", target, err)
	}

	if !info.Mode().IsRegular() {
		return nil, 0, fmt.Errorf("file %q is not a regular file", target)
	}

	data, err := root.ReadFile(filepath.FromSlash(target))
	if err != nil {
		return nil, 0, fmt.Errorf("read file %q: %w", target, err)
	}

	return data, info.Mode().Perm(), nil
}

func writeFile(root *os.Root, target string, data []byte, mode os.FileMode) error {
	file, err := root.OpenFile(
		filepath.FromSlash(target),
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		mode,
	)
	if err != nil {
		return fmt.Errorf("open file %q: %w", target, err)
	}

	if _, err := file.Write(data); err != nil {
		_ = file.Close()

		return fmt.Errorf("write file %q: %w", target, err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("close file %q: %w", target, err)
	}

	return nil
}

func safeRelativePath(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", errors.New("path must not be empty")
	}

	if strings.Contains(value, "\\") {
		return "", fmt.Errorf("path %q must use forward slashes", value)
	}

	cleaned := path.Clean(value)
	if cleaned != value || path.IsAbs(value) || !filepath.IsLocal(filepath.FromSlash(value)) {
		return "", fmt.Errorf("path %q must be a safe relative path", value)
	}

	return cleaned, nil
}

func safeJoin(root string, parts ...string) (string, error) {
	joined := root

	for _, part := range parts {
		cleaned, err := safeRelativePath(filepath.ToSlash(part))
		if err != nil {
			return "", err
		}

		joined = filepath.Join(joined, filepath.FromSlash(cleaned))
	}

	relative, err := filepath.Rel(root, joined)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes %q", joined, root)
	}

	return joined, nil
}

func appendUnique(values []string, value string) []string {
	if slices.Contains(values, value) {
		return values
	}

	return append(values, value)
}
