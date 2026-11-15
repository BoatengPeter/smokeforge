// Package selfupdate replaces the running smokeforge binary with a verified
// GitHub release asset.
package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/BoatengPeter/smokeforge/internal/version"
	"golang.org/x/mod/semver"
)

const (
	defaultAPIBase      = "https://api.github.com/repos/BoatengPeter/smokeforge"
	defaultDownloadBase = "https://github.com/BoatengPeter/smokeforge/releases/download"
	maxDownloadBytes    = 256 << 20
)

// Options controls a self-update request.
type Options struct {
	APIBase        string
	DownloadBase   string
	Executable     string
	CurrentVersion string
	GOOS           string
	GOARCH         string
	TargetVersion  string
	CheckOnly      bool
	Force          bool
	HTTPClient     *http.Client
}

// Result describes the resolved release and whether the binary changed.
type Result struct {
	Current    string `json:"current"`
	Latest     string `json:"latest"`
	Available  bool   `json:"available"`
	Updated    bool   `json:"updated"`
	Executable string `json:"executable"`
	Asset      string `json:"asset"`
}

type release struct {
	TagName string  `json:"tag_name"`
	Assets  []asset `json:"assets"`
}

type asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Update resolves a release, verifies its checksum, and atomically replaces the
// current executable. CheckOnly stops after resolving the release.
func Update(ctx context.Context, opts Options) (Result, error) {
	opts, err := normalizeOptions(opts)
	if err != nil {
		return Result{}, err
	}

	releaseInfo, err := resolveRelease(ctx, opts)
	if err != nil {
		return Result{}, err
	}

	result := makeResult(opts, releaseInfo.TagName)
	if opts.CheckOnly || (!result.Available && !opts.Force) {
		return result, nil
	}

	return installUpdate(ctx, opts, releaseInfo, result)
}

func normalizeOptions(opts Options) (Options, error) {
	if opts.TargetVersion == "" {
		opts.TargetVersion = "latest"
	}

	if opts.GOOS == "" {
		opts.GOOS = runtime.GOOS
	}

	if opts.GOARCH == "" {
		opts.GOARCH = runtime.GOARCH
	}

	if opts.Executable == "" {
		executable, err := os.Executable()
		if err != nil {
			return Options{}, fmt.Errorf("resolve executable: %w", err)
		}

		opts.Executable = executable
	}

	if opts.CurrentVersion == "" {
		opts.CurrentVersion = version.Current().Version
	}

	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: 2 * time.Minute}
	}

	return opts, nil
}

func makeResult(opts Options, latest string) Result {
	current := opts.CurrentVersion
	explicitVersion := opts.TargetVersion != "latest"

	available := explicitVersion && normalize(current) != normalize(latest)
	if !explicitVersion {
		available = current == "dev" ||
			semver.Compare(semver.Canonical(normalize(current)), semver.Canonical(normalize(latest))) < 0
	}

	return Result{
		Current:    current,
		Latest:     latest,
		Available:  available,
		Executable: opts.Executable,
	}
}

func installUpdate(
	ctx context.Context,
	opts Options,
	releaseInfo release,
	result Result,
) (Result, error) {
	assetName := archiveName(result.Latest, opts.GOOS, opts.GOARCH)

	asset, err := findAsset(releaseInfo, assetName)
	if err != nil {
		return Result{}, err
	}

	checksums, err := findAsset(releaseInfo, "checksums.txt")
	if err != nil {
		return Result{}, err
	}

	result.Asset = asset.Name

	checksumBody, err := download(ctx, opts.HTTPClient, assetURL(checksums, opts.DownloadBase, result.Latest))
	if err != nil {
		return Result{}, fmt.Errorf("download checksums: %w", err)
	}

	expected, err := checksumFor(checksumBody, asset.Name)
	if err != nil {
		return Result{}, err
	}

	archive, err := download(ctx, opts.HTTPClient, assetURL(asset, opts.DownloadBase, result.Latest))
	if err != nil {
		return Result{}, fmt.Errorf("download release asset: %w", err)
	}

	actual := sha256.Sum256(archive)
	if !bytes.Equal(actual[:], expected) {
		return Result{}, fmt.Errorf("checksum mismatch for %s", asset.Name)
	}

	if err := installArchive(opts.Executable, asset.Name, archive); err != nil {
		return Result{}, err
	}

	result.Updated = true

	return result, nil
}

func resolveRelease(ctx context.Context, opts Options) (release, error) {
	apiBase := strings.TrimRight(opts.APIBase, "/")
	if apiBase == "" {
		apiBase = defaultAPIBase
	}

	endpoint := apiBase + "/releases/latest"

	if opts.TargetVersion != "latest" {
		tag := opts.TargetVersion
		if !strings.HasPrefix(tag, "v") {
			tag = "v" + tag
		}

		endpoint = apiBase + "/releases/tags/" + tag
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return release{}, fmt.Errorf("build release request: %w", err)
	}

	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "smokeforge-self-update")

	response, err := opts.HTTPClient.Do(request)
	if err != nil {
		return release{}, fmt.Errorf("fetch release: %w", err)
	}

	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		return release{}, fmt.Errorf("fetch release: status %d", response.StatusCode)
	}

	var result release
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil {
		return release{}, fmt.Errorf("decode release: %w", err)
	}

	if result.TagName == "" {
		return release{}, errors.New("release has no tag")
	}

	return result, nil
}

func archiveName(tag, goos, goarch string) string {
	versionPart := strings.TrimPrefix(tag, "v")

	extension := ".tar.gz"
	if goos == "windows" {
		extension = ".zip"
	}

	return fmt.Sprintf("smokeforge_%s_%s_%s%s", versionPart, goos, goarch, extension)
}

func findAsset(releaseInfo release, name string) (asset, error) {
	for _, candidate := range releaseInfo.Assets {
		if candidate.Name == name {
			return candidate, nil
		}
	}

	return asset{}, fmt.Errorf("release asset %q not found", name)
}

func assetURL(asset asset, downloadBase, tag string) string {
	if asset.BrowserDownloadURL != "" {
		return asset.BrowserDownloadURL
	}

	base := strings.TrimRight(downloadBase, "/")
	if base == "" {
		base = defaultDownloadBase
	}

	return base + "/" + tag + "/" + asset.Name
}

func download(ctx context.Context, client *http.Client, rawURL string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build download request: %w", err)
	}

	request.Header.Set("User-Agent", "smokeforge-self-update")

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}

	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download: status %d", response.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxDownloadBytes))
	if err != nil {
		return nil, fmt.Errorf("read download: %w", err)
	}

	return body, nil
}

func checksumFor(body []byte, name string) ([]byte, error) {
	for line := range strings.SplitSeq(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || fields[1] != name {
			continue
		}

		decoded, err := hex.DecodeString(fields[0])
		if err != nil {
			return nil, fmt.Errorf("decode checksum for %s: %w", name, err)
		}

		return decoded, nil
	}

	return nil, fmt.Errorf("checksum for %s not found", name)
}

func installArchive(executable, name string, archive []byte) error {
	dir := filepath.Dir(executable)

	temp, err := os.CreateTemp(dir, ".smokeforge-update-*")
	if err != nil {
		return fmt.Errorf("create update file: %w", err)
	}

	tempName := temp.Name()

	defer func() {
		_ = os.Remove(tempName)
	}()

	if err := writeBinary(name, archive, temp); err != nil {
		_ = temp.Close()

		return err
	}

	if err := temp.Close(); err != nil {
		return fmt.Errorf("close update file: %w", err)
	}

	if err := os.Chmod(tempName, 0o755); err != nil { //nolint:gosec // installed binary must be executable
		return fmt.Errorf("make update executable: %w", err)
	}

	return replaceExecutable(executable, tempName)
}

func writeBinary(name string, archive []byte, output *os.File) error {
	if strings.HasSuffix(name, ".zip") {
		return writeZipBinary(archive, output)
	}

	return writeTarBinary(archive, output)
}

func writeZipBinary(archive []byte, output *os.File) error {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return fmt.Errorf("open update archive: %w", err)
	}

	for _, file := range reader.File {
		if filepath.Base(file.Name) != "smokeforge.exe" && filepath.Base(file.Name) != "smokeforge" {
			continue
		}

		source, err := file.Open()
		if err != nil {
			return fmt.Errorf("open update binary: %w", err)
		}

		_, copyErr := io.Copy(output, io.LimitReader(source, maxDownloadBytes))
		_ = source.Close()

		if copyErr != nil {
			return fmt.Errorf("extract update binary: %w", copyErr)
		}

		return nil
	}

	return errors.New("update archive has no smokeforge binary")
}

func writeTarBinary(archive []byte, output *os.File) error {
	reader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return fmt.Errorf("open update archive: %w", err)
	}

	defer func() {
		_ = reader.Close()
	}()

	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return fmt.Errorf("read update archive: %w", err)
		}

		if filepath.Base(header.Name) != "smokeforge" {
			continue
		}

		if _, err := io.Copy(output, io.LimitReader(tarReader, maxDownloadBytes)); err != nil {
			return fmt.Errorf("extract update binary: %w", err)
		}

		return nil
	}

	return errors.New("update archive has no smokeforge binary")
}

func replaceExecutable(executable, replacement string) error {
	backup := executable + ".old"
	_ = os.Remove(backup)

	if err := os.Rename(executable, backup); err != nil {
		return fmt.Errorf("prepare binary replacement: %w", err)
	}

	if err := os.Rename(replacement, executable); err != nil {
		_ = os.Rename(backup, executable)

		return fmt.Errorf("replace binary: %w", err)
	}

	_ = os.Remove(backup)

	return nil
}

func normalize(value string) string {
	if strings.HasPrefix(value, "v") {
		return value
	}

	return "v" + value
}
