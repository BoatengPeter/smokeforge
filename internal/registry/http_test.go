package registry

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
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

type testRegistry struct {
	server *httptest.Server
	cache  string
	opts   Options
}

func newTestRegistry(t *testing.T, checksum string) *testRegistry {
	t.Helper()

	archive := buildArchive(t, []archiveEntry{
		{name: "registry/manifest.yaml", body: remoteManifest, typeflag: tar.TypeReg},
	})
	sum := sha256.Sum256(archive)

	if checksum == "" {
		checksum = hex.EncodeToString(sum[:]) + "  registry.tar.gz\n"
	}

	mux := http.NewServeMux()
	mux.HandleFunc(
		"/releases/download/v1.0.0/registry.tar.gz",
		func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(archive)
		},
	)
	mux.HandleFunc(
		"/releases/download/v1.0.0/registry.sha256",
		func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, checksum)
		},
	)

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	cache := t.TempDir()

	return &testRegistry{
		server: server,
		cache:  cache,
		opts: Options{
			Source:   server.URL + "/releases",
			Tag:      "v1.0.0",
			CacheDir: cache,
		},
	}
}

func TestLoadRemoteCaching(t *testing.T) {
	t.Parallel()

	t.Run("downloads verifies and caches", func(t *testing.T) {
		t.Parallel()

		reg := newTestRegistry(t, "")

		first, err := Load(t.Context(), reg.opts)
		if err != nil {
			t.Fatalf("first load: %v", err)
		}

		if first.Manifest.Name != "remote" {
			t.Fatalf("name = %q, want remote", first.Manifest.Name)
		}

		if first.FromCache {
			t.Fatal("first load must not be served from cache")
		}

		offline := reg.opts
		offline.Offline = true

		second, err := Load(t.Context(), offline)
		if err != nil {
			t.Fatalf("offline load: %v", err)
		}

		if !second.FromCache {
			t.Fatal("offline load must be served from cache")
		}
	})

	t.Run("uses cache when online", func(t *testing.T) {
		t.Parallel()

		reg := newTestRegistry(t, "")
		if _, err := Load(t.Context(), reg.opts); err != nil {
			t.Fatal(err)
		}

		reg.server.Close()

		res, err := Load(t.Context(), reg.opts)
		if err != nil {
			t.Fatalf("cached load: %v", err)
		}

		if !res.FromCache {
			t.Fatal("expected FromCache to be true")
		}
	})
}

func TestLoadRemoteRefresh(t *testing.T) {
	t.Parallel()

	t.Run("refreshes remote registry", func(t *testing.T) {
		t.Parallel()

		reg := newTestRegistry(t, "")
		if _, err := Load(t.Context(), reg.opts); err != nil {
			t.Fatal(err)
		}

		res, err := Refresh(t.Context(), reg.opts)
		if err != nil {
			t.Fatalf("refresh: %v", err)
		}

		if res.FromCache {
			t.Fatal("refreshed registry must not be marked as cached")
		}
	})

	t.Run("redownloads when cache is corrupt", func(t *testing.T) {
		t.Parallel()

		reg := newTestRegistry(t, "")

		dir := entryDir(reg.cache, reg.opts.Source, reg.opts.Tag)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}

		path := filepath.Join(dir, "manifest.yaml")
		if err := os.WriteFile(path, []byte("not: valid"), 0o600); err != nil {
			t.Fatal(err)
		}

		res, err := Load(t.Context(), reg.opts)
		if err != nil {
			t.Fatalf("load: %v", err)
		}

		if res.Manifest.Name != "remote" {
			t.Fatalf("name = %q, want remote", res.Manifest.Name)
		}
	})
}

func TestLoadRemoteValidation(t *testing.T) {
	t.Parallel()

	t.Run("rejects checksum mismatch", func(t *testing.T) {
		t.Parallel()

		reg := newTestRegistry(t, strings.Repeat("0", 64)+"  registry.tar.gz\n")

		_, err := Load(t.Context(), reg.opts)
		if !errors.Is(err, ErrChecksumMismatch) {
			t.Fatalf("error = %v, want ErrChecksumMismatch", err)
		}
	})

	t.Run("rejects invalid tag before network access", func(t *testing.T) {
		t.Parallel()

		opts := Options{
			Source:   "http://127.0.0.1:1/releases",
			Tag:      "../escape",
			CacheDir: t.TempDir(),
		}

		_, err := Load(t.Context(), opts)
		if !errors.Is(err, ErrInvalidTag) {
			t.Fatalf("error = %v, want ErrInvalidTag", err)
		}
	})

	t.Run("rejects source without host", func(t *testing.T) {
		t.Parallel()

		opts := Options{
			Source:   "https:///releases",
			Tag:      "v1.0.0",
			CacheDir: t.TempDir(),
		}

		_, err := Load(t.Context(), opts)
		if !errors.Is(err, ErrUnsupportedSource) {
			t.Fatalf("error = %v, want ErrUnsupportedSource", err)
		}
	})

	t.Run("rejects credentials in source", func(t *testing.T) {
		t.Parallel()

		//nolint:gosec // test fixture exercising credential rejection.
		opts := Options{
			Source:   "https://user:secret@example.com/releases",
			Tag:      "v1.0.0",
			CacheDir: t.TempDir(),
		}

		_, err := Load(t.Context(), opts)
		if !errors.Is(err, ErrUnsupportedSource) {
			t.Fatalf("error = %v, want ErrUnsupportedSource", err)
		}
	})

	t.Run("rejects insecure remote source", func(t *testing.T) {
		t.Parallel()

		opts := Options{
			Source:   "http://example.com/releases",
			Tag:      "v1.0.0",
			CacheDir: t.TempDir(),
		}

		_, err := Load(t.Context(), opts)
		if !errors.Is(err, ErrUnsupportedSource) {
			t.Fatalf("error = %v, want ErrUnsupportedSource", err)
		}
	})

	t.Run("rejects insecure redirect", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "http://example.com/asset", http.StatusFound)
		}))
		t.Cleanup(server.Close)

		_, err := Load(t.Context(), Options{
			Source:   server.URL + "/releases",
			Tag:      "v1.0.0",
			CacheDir: t.TempDir(),
		})
		if !errors.Is(err, ErrUnsupportedSource) {
			t.Fatalf("error = %v, want ErrUnsupportedSource", err)
		}
	})

	t.Run("offline without cache fails", func(t *testing.T) {
		t.Parallel()

		opts := Options{
			Source:   "http://127.0.0.1:1/releases",
			Tag:      "v1.0.0",
			CacheDir: t.TempDir(),
			Offline:  true,
		}

		_, err := Load(t.Context(), opts)
		if !errors.Is(err, ErrOffline) {
			t.Fatalf("error = %v, want ErrOffline", err)
		}
	})
}

func TestLoadRemoteContext(t *testing.T) {
	t.Parallel()

	reg := newTestRegistry(t, "")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := Load(ctx, reg.opts)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestCheckRedirect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url     string
		via     int
		wantErr bool
	}{
		{name: "https allowed", url: "https://example.com/asset", via: 1},
		{name: "http loopback allowed", url: "http://127.0.0.1:8080/asset", via: 1},
		{name: "http rejected", url: "http://example.com/asset", via: 1, wantErr: true},
		//nolint:gosec // test fixture exercising credential rejection.
		{name: "credentials rejected", url: "https://user:secret@example.com/asset", via: 1, wantErr: true},
		{name: "too many redirects", url: "https://example.com/asset", via: maxRedirects, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tt.url, nil)
			via := make([]*http.Request, tt.via)

			err := checkRedirect(req, via)
			if tt.wantErr && err == nil {
				t.Fatal("expected redirect error")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestReleaseURLLatest(t *testing.T) {
	t.Parallel()

	got := releaseURL(Options{
		Source: "https://example.com/releases/",
		Tag:    "latest",
	}, "registry.tar.gz")

	want := "https://example.com/releases/latest/download/registry.tar.gz"
	if got != want {
		t.Fatalf("url = %q, want %q", got, want)
	}
}
