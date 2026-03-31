package image

import (
	"bytes"
	"image"
	"image/jpeg"

	"github.com/disintegration/imaging"
	"github.com/sirupsen/logrus"
)

const (
	// ContrastLevel defines the level of contrast enhancement to be applied to the image.
	ContrastLevel = 25.0
	// SharpenLevel defines the level of sharpening to be applied to the image.
	SharpenLevel = 1.0
	// QualityLevel defines the quality level for JPEG encoding (0-100).
	QualityLevel = 70
)

// Converter is a struct that provides methods for converting image payloads into enhanced grayscale images.
type Converter struct{}

// NewConverter creates a new instance of the Converter struct, which can be used to convert image payloads into enhanced grayscale images.
func NewConverter() *Converter {
	return &Converter{}
}

// FromPayload takes a byte slice representing an image payload, enhances the contrast and converts it to grayscale, and returns the resulting image.Image object.
// payload: The byte slice containing the image data to be converted.
func (c *Converter) FromPayload(payload []byte) (image.Image, error) {
	logrus.Debugf("converting image payload of size: %d bytes", len(payload))

	src, err := imaging.Decode(bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	// img := imaging.Grayscale(src)
	img := imaging.AdjustContrast(src, ContrastLevel)
	img = imaging.Sharpen(img, SharpenLevel)

	logrus.Debugf(
		"converted image: original size=%dx%d, processed size=%dx%d",
		src.Bounds().Dx(),
		src.Bounds().Dy(),
		img.Bounds().Dx(),
		img.Bounds().Dy(),
	)

	return img, nil
}

// Encode takes an image.Image object and encodes it into a byte slice in JPEG format with a specified quality level.
// image: The image.Image object to be encoded.
func (c *Converter) Encode(image image.Image) ([]byte, error) {
	logrus.Debugf("encoding image: size=%dx%d", image.Bounds().Dx(), image.Bounds().Dy())

	var buf bytes.Buffer
	err := jpeg.Encode(&buf, image, &jpeg.Options{Quality: QualityLevel})

	logrus.Debugf("encoded image: size=%d bytes", len(buf.Bytes()))

	return buf.Bytes(), err
}
