package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/muhlba91/watermeter-image-processor/cmd/configuration"
)

// dirPermissions is the permission mode used when creating storage directories.
const dirPermissions = 0o750

// filePermissions is the permission mode used when writing image files.
const filePermissions = 0o640

// Writer is a storage.Writer implementation that writes images to a local filesystem path.
type Writer struct {
	// path is the base directory processed images are written under.
	path string
}

// NewWriter creates a new instance of Writer.
// configuration: The configuration data required to determine the storage path.
func NewWriter(configuration *configuration.Data) (*Writer, error) {
	return &Writer{
		path: fmt.Sprintf(configuration.FileStoragePath, configuration.MeterID),
	}, nil
}

// HealthCheck verifies that the configured storage directory exists and is writable.
func (f *Writer) HealthCheck() bool {
	return os.MkdirAll(f.path, dirPermissions) == nil
}

// Write writes the given data to a timestamped file under the configured storage path.
// ctx: Unused; present to satisfy the storage.Writer interface.
// data: The raw image data to write.
func (f *Writer) Write(_ context.Context, data []byte) {
	now := time.Now()
	dir := filepath.Join(f.path, now.Format("2006/01/02"))

	if err := os.MkdirAll(dir, dirPermissions); err != nil {
		logrus.Errorf("failed to create local storage directory %q: %v", dir, err)
		return
	}

	path := filepath.Join(dir, now.Format("15-04-05")+".jpg")

	logrus.Debugf("writing image to local storage: %s (size=%d bytes)", path, len(data))
	if err := os.WriteFile(path, data, filePermissions); err != nil {
		logrus.Errorf("failed to write image to local storage %q: %v", path, err)
		return
	}

	logrus.Infof("wrote image to local storage: %s", path)
}
