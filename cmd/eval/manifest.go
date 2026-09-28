package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
)

// manifestFile is the file name of the test image manifest inside the image directory.
const manifestFile = "manifest.json"

// testImage describes a single test image and the readings accepted as correct for it.
type testImage struct {
	// File is the image path, relative to the manifest's directory.
	File string `json:"file"`
	// Accept lists the readings accepted as correct, e.g. "616.60", compared numerically.
	Accept []string `json:"accept"`
	// Note describes the image and what it tests.
	Note string `json:"note,omitempty"`
}

// manifest is the list of test images in an image directory.
type manifest struct {
	// Images are the test images, sorted by file name.
	Images []testImage `json:"images"`
}

// loadManifest reads the manifest in dir, returning an empty manifest if none exists yet.
// dir: The image directory.
func loadManifest(dir string) (*manifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, manifestFile))
	if errors.Is(err, os.ErrNotExist) {
		return &manifest{}, nil
	}
	if err != nil {
		return nil, err
	}

	var m manifest
	if uErr := json.Unmarshal(data, &m); uErr != nil {
		return nil, uErr
	}
	return &m, nil
}

// upsert adds img to the manifest, or updates the accepted readings of an existing entry for the
// same file while keeping its note, so hand-written descriptions survive regenerating the images.
// img: The test image entry.
func (m *manifest) upsert(img testImage) {
	for i := range m.Images {
		if m.Images[i].File == img.File {
			m.Images[i].Accept = img.Accept
			return
		}
	}
	m.Images = append(m.Images, img)
	sort.Slice(m.Images, func(i, j int) bool { return m.Images[i].File < m.Images[j].File })
}

// save writes the manifest to dir.
// dir: The image directory.
func (m *manifest) save(dir string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, manifestFile), append(data, '\n'), 0o600)
}
