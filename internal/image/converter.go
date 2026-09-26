package image

import (
	"bytes"
	"image"
	"image/jpeg"

	"github.com/disintegration/imaging"
	"github.com/sirupsen/logrus"
)

const (
	// GammaLevel defines the level of gamma adjustment to be applied to the image for shadow lifting.
	GammaLevel = 2.0
	// ContrastLevel defines the level of contrast enhancement to be applied to the image.
	ContrastLevel = 20.0
	// SharpenLevel defines the level of sharpening to be applied to the image.
	SharpenLevel = 0.5
	// QualityLevel defines the quality level for JPEG encoding (0-100).
	QualityLevel = 85

	// rgba64To8BitShift converts an image/color.RGBA64's 16-bit (0-65535) channel value down to the
	// 8-bit (0-255) range returned by color.RGBA.
	rgba64To8BitShift = 8

	// redMinChannel is the minimum 8-bit red channel value for a pixel to be considered part of the
	// red decimal digit wheels, filtering out generally dark background pixels.
	redMinChannel = 30
	// redOverGreenMargin and redOverBlueMargin are how far the red channel must exceed the green and
	// blue channels for a pixel to be classified as reddish, rather than a neutral gray/white pixel.
	redOverGreenMargin = 15
	redOverBlueMargin  = 10
	// minRedAreaFraction is the minimum fraction of the image area the detected red-hued region must
	// cover to be trusted as the illuminated red digit wheels, rather than a stray reddish pixel or
	// JPEG compression artifact.
	minRedAreaFraction = 0.0005
	// maxRedAreaFraction is the maximum fraction of the image area the detected red-hued region may
	// cover; anything larger is treated as red ambient lighting or a red-tinted photo rather than the
	// small red digit wheels.
	maxRedAreaFraction = 0.2
	// blackToRedWidthRatio infers the width of the black (whole cubic-meter) digit wheels from the
	// detected red (decimal) wheel width, assuming all wheels on the counter are the same physical
	// size: it mirrors the 5 black : 2 red digit split the AI prompt assumes (internal/image/ai).
	blackToRedWidthRatio = 5.0 / 2.0
	// roiMarginFraction expands the inferred digit-strip bounding box by this fraction on each side,
	// so digit edges are not clipped by an overly tight detection.
	roiMarginFraction = 0.15
	// roiTargetHeight is the minimum height, in pixels, the cropped region of interest is upscaled
	// to before enhancement, so small digit wheels get more effective resolution for the AI models.
	roiTargetHeight = 240
)

// Converter is a struct that provides methods for converting image payloads into enhanced images.
type Converter struct {
	// roiCropEnabled controls whether FromPayload attempts to crop to the detected digit strip
	// before enhancement, or always uses the full image.
	roiCropEnabled bool
}

// NewConverter creates a new instance of the Converter struct, which can be used to convert image payloads into enhanced images.
// roiCropEnabled: Whether to crop incoming images to the detected digit strip before enhancement.
func NewConverter(roiCropEnabled bool) *Converter {
	return &Converter{roiCropEnabled: roiCropEnabled}
}

// FromPayload takes a byte slice representing an image payload, crops it to the detected digit
// window when possible, lifts shadows with gamma adjustment, enhances contrast and sharpness, and
// returns the resulting image.Image object.
// payload: The byte slice containing the image data to be converted.
func (c *Converter) FromPayload(payload []byte) (image.Image, error) {
	logrus.Debugf("converting image payload of size: %d bytes", len(payload))

	src, err := imaging.Decode(bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	roi := src
	if !c.roiCropEnabled {
		logrus.Debugf("digit strip roi cropping is disabled, using full image")
	} else if rect := detectDigitStripROI(src); rect != nil {
		roi = imaging.Crop(src, *rect)
		if roi.Bounds().Dy() > 0 && roi.Bounds().Dy() < roiTargetHeight {
			scale := float64(roiTargetHeight) / float64(roi.Bounds().Dy())
			roi = imaging.Resize(
				roi,
				int(float64(roi.Bounds().Dx())*scale),
				roiTargetHeight,
				imaging.Lanczos,
			)
		}
		logrus.Debugf("detected digit strip roi: %v", *rect)
	} else {
		logrus.Debugf("no digit strip roi detected, using full image")
	}

	img := imaging.AdjustGamma(roi, GammaLevel)
	img = imaging.AdjustContrast(img, ContrastLevel)
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

// detectDigitStripROI locates the red decimal digit wheels by their distinctive hue, which stays
// identifiable even in a very dark or unevenly lit photo where absolute brightness thresholding does
// not reliably separate the digit window from the rest of the meter body. It then infers the
// bounding box of the full digit strip (black wheels plus red wheels) from the known wheel-width
// ratio, expanded by a margin. It returns nil if no plausible red region was found, in which case the
// caller should fall back to using the full image.
// img: The source image to search for the red digit wheels in.
func detectDigitStripROI(img image.Image) *image.Rectangle {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width == 0 || height == 0 {
		return nil
	}

	mask := make([][]bool, height)
	for y := range height {
		mask[y] = make([]bool, width)
		for x := range width {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			r8 := int(r >> rgba64To8BitShift)
			g8 := int(g >> rgba64To8BitShift)
			b8 := int(b >> rgba64To8BitShift)
			mask[y][x] = r8 > redMinChannel && r8 > g8+redOverGreenMargin && r8 > b8+redOverBlueMargin
		}
	}

	red := largestRegion(mask, width, height)
	if red == nil {
		return nil
	}

	areaFraction := float64(red.Dx()*red.Dy()) / float64(width*height)
	if areaFraction < minRedAreaFraction || areaFraction > maxRedAreaFraction {
		return nil
	}

	blackWidth := int(float64(red.Dx()) * blackToRedWidthRatio)
	strip := image.Rect(red.Min.X-blackWidth, red.Min.Y, red.Max.X, red.Max.Y)

	marginX := int(float64(strip.Dx()) * roiMarginFraction)
	marginY := int(float64(strip.Dy()) * roiMarginFraction)
	expanded := image.Rect(
		max(0, strip.Min.X-marginX),
		max(0, strip.Min.Y-marginY),
		min(width, strip.Max.X+marginX),
		min(height, strip.Max.Y+marginY),
	).Add(bounds.Min)

	return &expanded
}

// point represents a pixel coordinate used while flood-filling connected regions.
type point struct {
	x, y int
}

// largestRegion finds the bounding rectangle of the largest 4-connected group of true pixels in
// mask, using an iterative flood fill.
// mask: A width x height grid where true marks a pixel of interest, indexed [y][x].
// width: The image width in pixels.
// height: The image height in pixels.
func largestRegion(mask [][]bool, width, height int) *image.Rectangle {
	visited := make([][]bool, height)
	for y := range visited {
		visited[y] = make([]bool, width)
	}

	var best image.Rectangle
	bestArea := 0

	for y := range height {
		for x := range width {
			if visited[y][x] || !mask[y][x] {
				continue
			}

			visited[y][x] = true
			rect, area := floodFill(mask, visited, width, height, point{x, y})
			if area > bestArea {
				bestArea = area
				best = rect
			}
		}
	}

	if bestArea == 0 {
		return nil
	}
	return &best
}

// floodFill explores the 4-connected true region of mask starting at start, marking visited pixels
// along the way, and returns its bounding rectangle and pixel area.
// mask: A width x height grid where true marks a pixel of interest, indexed [y][x].
// visited: Tracks which pixels have already been assigned to a region; updated in place.
// width: The image width in pixels.
// height: The image height in pixels.
// start: The seed pixel to begin the flood fill from; must already be marked visited by the caller.
func floodFill(mask [][]bool, visited [][]bool, width, height int, start point) (image.Rectangle, int) {
	minX, minY, maxX, maxY, area := start.x, start.y, start.x, start.y, 0
	queue := []point{start}

	for len(queue) > 0 {
		p := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		area++
		minX, maxX = min(minX, p.x), max(maxX, p.x)
		minY, maxY = min(minY, p.y), max(maxY, p.y)

		for _, n := range [4]point{{p.x - 1, p.y}, {p.x + 1, p.y}, {p.x, p.y - 1}, {p.x, p.y + 1}} {
			if n.x < 0 || n.x >= width || n.y < 0 || n.y >= height || visited[n.y][n.x] || !mask[n.y][n.x] {
				continue
			}
			visited[n.y][n.x] = true
			queue = append(queue, n)
		}
	}

	return image.Rect(minX, minY, maxX+1, maxY+1), area
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
