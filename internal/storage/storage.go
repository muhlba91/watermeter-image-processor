package storage

import (
	"fmt"
	"sort"
	"strings"

	"github.com/muhlba91/watermeter-image-processor/cmd/configuration"
	"github.com/muhlba91/watermeter-image-processor/internal/storage/file"
	"github.com/muhlba91/watermeter-image-processor/internal/storage/scaleway/object"
)

// ProviderFile is the identifier for the local filesystem storage provider.
const ProviderFile = "file"

// ProviderScaleway is the identifier for the Scaleway S3-compatible object storage provider.
const ProviderScaleway = "scaleway"

// providerFactory is a function type that constructs a Writer from configuration.
type providerFactory func(*configuration.Data) (Writer, error)

// registeredProviders returns the registry of all supported storage providers.
func registeredProviders() map[string]providerFactory {
	return map[string]providerFactory{
		ProviderFile: func(cfg *configuration.Data) (Writer, error) {
			return file.NewWriter(cfg)
		},
		ProviderScaleway: func(cfg *configuration.Data) (Writer, error) {
			return object.NewUploader(cfg)
		},
	}
}

// NewProvider instantiates the Writer implementation selected by configuration.StorageProvider.
func NewProvider(cfg *configuration.Data) (Writer, error) {
	registry := registeredProviders()

	factory, ok := registry[cfg.StorageProvider]
	if !ok {
		return nil, fmt.Errorf(
			"unknown storage provider %q: must be one of [%s]",
			cfg.StorageProvider,
			supportedProviders(registry),
		)
	}

	return factory(cfg)
}

// supportedProviders returns a sorted, comma-separated list of provider identifiers from the given registry.
func supportedProviders(registry map[string]providerFactory) string {
	keys := make([]string, 0, len(registry))
	for k := range registry {
		keys = append(keys, fmt.Sprintf("%q", k))
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}
