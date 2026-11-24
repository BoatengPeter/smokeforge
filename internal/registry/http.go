package registry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	archiveFileName     = "registry.tar.gz"
	checksumFileName    = "registry.sha256"
	httpTimeout         = 2 * time.Minute
	maxArchiveBytes     = 512 << 20
	maxChecksumFileSize = 4096
	maxRedirects        = 10
)

var tagPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type remoteTarget struct {
	options Options
	dir     string
}

func prepareRemote(opts Options) (*remoteTarget, error) {
	source := opts.Source
	if source == "" {
		source = DefaultURL
	}

	opts.Source = source

	if opts.Tag == "" {
		opts.Tag = "latest"
	}

	if !tagPattern.MatchString(opts.Tag) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidTag, opts.Tag)
	}

	if err := validateSource(source); err != nil {
		return nil, err
	}

	cacheDir := opts.CacheDir
	if cacheDir == "" {
		var err error

		cacheDir, err = DefaultCacheDir()
		if err != nil {
			return nil, err
		}
	}

	return &remoteTarget{
		options: opts,
		dir:     entryDir(cacheDir, source, opts.Tag),
	}, nil
}

func loadRemote(ctx context.Context, opts Options) (*Result, error) {
	target, err := prepareRemote(opts)
	if err != nil {
		return nil, err
	}

	cached, cacheErr := resultFromManifest(target.options, target.dir, true)
	if cacheErr == nil && cacheUsable(target.options, target.dir) {
		return cached, nil
	}

	if target.options.Offline {
		if cacheErr == nil {
			return cached, nil
		}

		return nil, fmt.Errorf("%w: %w", ErrOffline, cacheErr)
	}

	if err := download(ctx, target.options, target.dir); err != nil {
		return nil, err
	}

	if err := writeCacheVersion(target.dir); err != nil {
		return nil, err
	}

	return resultFromManifest(target.options, target.dir, false)
}

// Refresh forces a fresh download of a remote registry and replaces its cache
// entry. A local registry is revalidated and returned unchanged. Offline
// refreshes are rejected.
func Refresh(ctx context.Context, opts Options) (*Result, error) {
	source := opts.Source
	if source == "" {
		source = DefaultURL
	}

	if isLocalSource(source) {
		return loadLocal(source, opts.CLIVersion)
	}

	opts.Source = source

	target, err := prepareRemote(opts)
	if err != nil {
		return nil, err
	}

	if target.options.Offline {
		return nil, fmt.Errorf("%w: cannot refresh registry", ErrOffline)
	}

	if err := os.RemoveAll(target.dir); err != nil {
		return nil, fmt.Errorf("clear registry cache: %w", err)
	}

	if err := download(ctx, target.options, target.dir); err != nil {
		return nil, err
	}

	if err := writeCacheVersion(target.dir); err != nil {
		return nil, err
	}

	return resultFromManifest(target.options, target.dir, false)
}

func resultFromManifest(opts Options, dir string, fromCache bool) (*Result, error) {
	m, err := loadManifest(filepath.Join(dir, "manifest.yaml"), opts.CLIVersion)
	if err != nil {
		return nil, fmt.Errorf("load cached registry: %w", err)
	}

	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("validate cached registry: %w", err)
	}

	return &Result{
		Manifest:  m,
		Source:    opts.Source,
		Tag:       opts.Tag,
		Dir:       dir,
		FromCache: fromCache,
	}, nil
}

func validateSource(source string) error {
	parsed, err := url.Parse(source)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnsupportedSource, err)
	}

	if parsed.Host == "" {
		return fmt.Errorf("%w: %q has no host", ErrUnsupportedSource, source)
	}

	if parsed.User != nil {
		return fmt.Errorf("%w: credentials are not allowed in %q", ErrUnsupportedSource, source)
	}

	switch parsed.Scheme {
	case "https":
		return nil
	case "http":
		if isLoopbackHost(parsed.Hostname()) {
			return nil
		}

		return fmt.Errorf("%w: %q must use https", ErrUnsupportedSource, source)
	default:
		return fmt.Errorf("%w: %q", ErrUnsupportedSource, source)
	}
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}

	ip := net.ParseIP(host)

	return ip != nil && ip.IsLoopback()
}

func download(ctx context.Context, opts Options, dir string) error {
	client := newHTTPClient()

	want, err := fetchChecksum(ctx, client, releaseURL(opts, checksumFileName))
	if err != nil {
		return err
	}

	parent := filepath.Dir(dir)
	if err := os.MkdirAll(parent, 0o750); err != nil {
		return fmt.Errorf("create registry cache: %w", err)
	}

	tempDir, err := os.MkdirTemp(parent, ".download-*")
	if err != nil {
		return fmt.Errorf("create registry staging directory: %w", err)
	}

	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	archivePath := filepath.Join(tempDir, archiveFileName)

	got, err := fetchArchive(ctx, client, releaseURL(opts, archiveFileName), archivePath)
	if err != nil {
		return err
	}

	if got != want {
		return fmt.Errorf("%w: got %s, want %s", ErrChecksumMismatch, got, want)
	}

	extracted := filepath.Join(tempDir, "registry")
	if err := extractArchive(ctx, archivePath, extracted); err != nil {
		return err
	}

	contentRoot, err := archiveContentRoot(extracted)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("clear stale registry cache: %w", err)
	}

	if err := os.Rename(contentRoot, dir); err != nil {
		return fmt.Errorf("commit registry cache: %w", err)
	}

	return nil
}

func releaseURL(opts Options, name string) string {
	base := strings.TrimSuffix(opts.Source, "/")
	if opts.Tag == "latest" {
		return base + "/latest/download/" + name
	}

	return base + "/download/" + url.PathEscape(opts.Tag) + "/" + name
}

func fetchChecksum(ctx context.Context, client *http.Client, source string) (string, error) {
	body, err := fetch(ctx, client, source)
	if err != nil {
		return "", err
	}

	data, readErr := io.ReadAll(io.LimitReader(body, maxChecksumFileSize))
	closeErr := body.Close()

	if readErr != nil {
		return "", fmt.Errorf("read checksum from %q: %w", source, readErr)
	}

	if closeErr != nil {
		return "", fmt.Errorf("close checksum response from %q: %w", source, closeErr)
	}

	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return "", fmt.Errorf("invalid checksum file from %q", source)
	}

	sum := strings.ToLower(fields[0])
	if _, err := hex.DecodeString(sum); err != nil || len(sum) != sha256.Size*2 {
		return "", fmt.Errorf("invalid checksum %q from %q", fields[0], source)
	}

	return sum, nil
}

func fetchArchive(
	ctx context.Context,
	client *http.Client,
	source string,
	path string,
) (string, error) {
	body, err := fetch(ctx, client, source)
	if err != nil {
		return "", err
	}

	//nolint:gosec // path is a staging file created under the validated cache directory.
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		_ = body.Close()

		return "", fmt.Errorf("create archive file: %w", err)
	}

	hasher := sha256.New()
	limited := io.LimitReader(body, maxArchiveBytes+1)
	written, copyErr := io.Copy(io.MultiWriter(file, hasher), limited)
	closeBodyErr := body.Close()
	closeFileErr := file.Close()

	if copyErr != nil {
		return "", fmt.Errorf("download registry archive: %w", copyErr)
	}

	if closeBodyErr != nil {
		return "", fmt.Errorf("close registry response: %w", closeBodyErr)
	}

	if closeFileErr != nil {
		return "", fmt.Errorf("close registry archive: %w", closeFileErr)
	}

	if written > maxArchiveBytes {
		return "", fmt.Errorf("%w: archive exceeds %d bytes", ErrUnsafeArchive, maxArchiveBytes)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func newHTTPClient() *http.Client {
	return &http.Client{
		Timeout:       httpTimeout,
		CheckRedirect: checkRedirect,
	}
}

func checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return fmt.Errorf("stopped after %d redirects", maxRedirects)
	}

	return validateSource(req.URL.String())
}

func fetch(ctx context.Context, client *http.Client, source string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return nil, fmt.Errorf("build registry request for %q: %w", source, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch registry from %q: %w", source, err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()

		return nil, fmt.Errorf("fetch registry from %q: unexpected status %s", source, resp.Status)
	}

	return resp.Body, nil
}

func extractArchive(ctx context.Context, path, dst string) error {
	//nolint:gosec // path is a staging file created under the validated cache directory.
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open registry archive: %w", err)
	}

	extractErr := Extract(ctx, file, dst)
	closeErr := file.Close()

	if extractErr != nil {
		return extractErr
	}

	if closeErr != nil {
		return fmt.Errorf("close registry archive: %w", closeErr)
	}

	return nil
}

func archiveContentRoot(extracted string) (string, error) {
	if isFile(filepath.Join(extracted, "manifest.yaml")) {
		return extracted, nil
	}

	nested := filepath.Join(extracted, "registry")
	if isFile(filepath.Join(nested, "manifest.yaml")) {
		return nested, nil
	}

	return "", errors.New("registry archive is missing manifest.yaml")
}

func isFile(path string) bool {
	info, err := os.Stat(path)

	return err == nil && info.Mode().IsRegular()
}
