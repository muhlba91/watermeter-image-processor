package storage

import "context"

// Writer defines the interface for persisting processed images to a storage backend. It is a
// best-effort side effect: implementations must not return an error, and must log and swallow any
// failure rather than block or fail image processing.
type Writer interface {
	// HealthCheck verifies the overall health of the storage backend.
	HealthCheck() bool
	// Write persists the given image data to the storage backend.
	// ctx: The context for the operation.
	// data: The raw image data to persist.
	Write(ctx context.Context, data []byte)
}
