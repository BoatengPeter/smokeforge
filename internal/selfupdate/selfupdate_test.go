package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateDownloadsVerifiesAndReplaces(t *testing.T) {
	t.Parallel()

	archive := testArchive(t, []byte("new-binary"))
	assetName := "smokeforge_0.1.6_linux_amd64.tar.gz"
	checksums := checksumLine(archive, assetName)

	server := testReleaseServer(t, "v0.1.6", assetName, archive, checksums)
	defer server.Close()

	executable := filepath.Join(t.TempDir(), "smokeforge")
	if err := os.WriteFile(executable, []byte("old-binary"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := Update(t.Context(), testOptions(server.URL, executable, "v0.1.5"))
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	if !result.Updated || !result.Available {
		t.Fatalf("result = %#v", result)
	}

	body, err := os.ReadFile(executable) //nolint:gosec // test temp file
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != "new-binary" {
		t.Fatalf("binary = %q", body)
	}
}

func TestUpdateCheckOnly(t *testing.T) {
	t.Parallel()

	server := testReleaseServer(t, "v0.1.6", "smokeforge_0.1.6_linux_amd64.tar.gz", nil, "")
	defer server.Close()

	executable := filepath.Join(t.TempDir(), "smokeforge")
	if err := os.WriteFile(executable, []byte("old-binary"), 0o600); err != nil {
		t.Fatal(err)
	}

	options := testOptions(server.URL, executable, "v0.1.5")
	options.CheckOnly = true

	result, err := Update(t.Context(), options)
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	if !result.Available || result.Updated {
		t.Fatalf("result = %#v", result)
	}

	body, _ := os.ReadFile(executable) //nolint:gosec // test temp file
	if string(body) != "old-binary" {
		t.Fatalf("binary changed: %q", body)
	}
}

func TestUpdateRejectsChecksumMismatch(t *testing.T) {
	t.Parallel()

	archive := testArchive(t, []byte("new-binary"))
	assetName := "smokeforge_0.1.6_linux_amd64.tar.gz"

	server := testReleaseServer(t, "v0.1.6", assetName, archive, "deadbeef  "+assetName+"\n")
	defer server.Close()

	executable := filepath.Join(t.TempDir(), "smokeforge")
	if err := os.WriteFile(executable, []byte("old-binary"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Update(t.Context(), testOptions(server.URL, executable, "v0.1.5")); err == nil {
		t.Fatal("expected checksum error")
	}
}

func TestUpdateAlreadyCurrent(t *testing.T) {
	t.Parallel()

	server := testReleaseServer(t, "v0.1.5", "smokeforge_0.1.5_linux_amd64.tar.gz", nil, "")
	defer server.Close()

	result, err := Update(t.Context(), testOptions(server.URL, filepath.Join(t.TempDir(), "smokeforge"), "v0.1.5"))
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	if result.Available || result.Updated {
		t.Fatalf("result = %#v", result)
	}
}

func TestUpdateExplicitOlderVersion(t *testing.T) {
	t.Parallel()

	archive := testArchive(t, []byte("older-binary"))
	assetName := "smokeforge_0.1.4_linux_amd64.tar.gz"

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/releases/tags/v0.1.4" {
			writeRelease(t, writer, serverURL(request), "v0.1.4", assetName, archive, checksumLine(archive, assetName))

			return
		}

		if request.URL.Path == "/"+assetName {
			_, _ = writer.Write(archive)

			return
		}

		if request.URL.Path == "/checksums.txt" {
			_, _ = writer.Write([]byte(checksumLine(archive, assetName)))

			return
		}

		http.NotFound(writer, request)
	}))
	defer server.Close()

	executable := filepath.Join(t.TempDir(), "smokeforge")
	if err := os.WriteFile(executable, []byte("old-binary"), 0o600); err != nil {
		t.Fatal(err)
	}

	options := testOptions(server.URL, executable, "v0.1.5")
	options.TargetVersion = "v0.1.4"

	result, err := Update(t.Context(), options)
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	if !result.Updated || result.Latest != "v0.1.4" {
		t.Fatalf("result = %#v", result)
	}
}

func testOptions(serverURL, executable, current string) Options {
	return Options{
		APIBase:        serverURL,
		DownloadBase:   serverURL,
		Executable:     executable,
		CurrentVersion: current,
		GOOS:           "linux",
		GOARCH:         "amd64",
		HTTPClient:     http.DefaultClient,
	}
}

func testReleaseServer(t *testing.T, tag, assetName string, archive []byte, checksums string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/releases/latest" {
			writeRelease(t, writer, serverURL(request), tag, assetName, archive, checksums)

			return
		}

		if request.URL.Path == "/"+assetName {
			_, _ = writer.Write(archive)

			return
		}

		if request.URL.Path == "/checksums.txt" {
			_, _ = writer.Write([]byte(checksums))

			return
		}

		http.NotFound(writer, request)
	}))
}

func writeRelease(t *testing.T, writer http.ResponseWriter, base, tag, assetName string, archive []byte, checksums string) {
	t.Helper()

	_ = checksums

	assets := []map[string]string{{"name": assetName, "browser_download_url": base + "/" + assetName}}
	if archive != nil {
		assets = append(assets, map[string]string{"name": "checksums.txt", "browser_download_url": base + "/checksums.txt"})
	}

	payload := map[string]any{"tag_name": tag, "assets": assets}

	writer.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(writer).Encode(payload); err != nil {
		t.Errorf("encode release: %v", err)
	}
}

func serverURL(request *http.Request) string {
	scheme := "http"
	if request.TLS != nil {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s", scheme, request.Host)
}

func checksumLine(archive []byte, name string) string {
	sum := sha256.Sum256(archive)

	return hex.EncodeToString(sum[:]) + "  " + name + "\n"
}

func testArchive(t *testing.T, binary []byte) []byte {
	t.Helper()

	var buffer bytes.Buffer

	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)

	if err := tarWriter.WriteHeader(&tar.Header{Name: "smokeforge", Mode: 0o755, Size: int64(len(binary))}); err != nil {
		t.Fatal(err)
	}

	if _, err := tarWriter.Write(binary); err != nil {
		t.Fatal(err)
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}

	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}

	return buffer.Bytes()
}
