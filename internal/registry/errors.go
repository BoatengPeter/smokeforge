package registry

import "errors"

// Errors returned by registry loading. Use errors.Is to inspect them.
var (
	// ErrInvalidTag indicates a registry tag contains unsafe characters.
	ErrInvalidTag = errors.New("invalid registry tag")
	// ErrOffline indicates the registry is unavailable in the cache while
	// offline mode is enabled.
	ErrOffline = errors.New("registry unavailable in offline mode")
	// ErrChecksumMismatch indicates the downloaded archive does not match the
	// published SHA-256 checksum.
	ErrChecksumMismatch = errors.New("registry checksum mismatch")
	// ErrUnsafeArchive indicates an archive entry escapes the extraction root
	// or uses an unsupported type.
	ErrUnsafeArchive = errors.New("unsafe registry archive")
	// ErrUnsupportedSource indicates the registry source URL uses a scheme or
	// host that smokeforge does not fetch from.
	ErrUnsupportedSource = errors.New("unsupported registry source")
)
