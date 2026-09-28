package image

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"

	"github.com/disintegration/imaging"
	"github.com/sirupsen/logrus"
)

const (
	// deblockSigma is the Gaussian blur sigma, in source pixels, used to smooth out the 8x8 block
	// edges of heavily compressed JPEG frames, so they are not mistaken for digit strokes and are not
	// magnified by the upscaled digit strip crop.
	deblockSigma = 0.8
	// levelsLowPercentile and levelsHighPercentile are the luminance percentiles stretched to black
	// and white respectively, ignoring a few outlier pixels such as specular highlights.
	levelsLowPercentile  = 0.01
	levelsHighPercentile = 0.99
	// maxChannel is the maximum 8-bit channel value.
	maxChannel = 255
	// histogramBins is the number of bins in an 8-bit luminance histogram.
	histogramBins = 256
	// lumaR, lumaG and lumaB are the ITU-R BT.601 luma weights.
	lumaR = 0.299
	lumaG = 0.587
	lumaB = 0.114

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
)

// Converter is a struct that provides methods for converting image payloads into enhanced images.
type Converter struct {
	// rotationDegrees is the counter-clockwise rotation applied to every incoming image, to level a
	// camera that is not mounted straight.
	rotationDegrees float64
}

// NewConverter creates a new instance of the Converter struct, which can be used to convert image payloads into enhanced images.
// rotationDegrees: The counter-clockwise rotation, in degrees, applied to every incoming image (0 to disable).
func NewConverter(rotationDegrees float64) *Converter {
	return &Converter{rotationDegrees: rotationDegrees}
}

// FromPayload decodes an image payload and prepares it for the AI provider: it levels the image by
// the configured rotation, smooths out JPEG block artifacts, and stretches the image's own
// brightness range, which lifts dim frames without the posterization of a fixed gamma/contrast
// boost. If the digit strip can be located, an enlarged crop of it is stacked below the full image:
// the models read best with both the full meter for context and the enlarged digits for detail,
// while a crop alone loses that context whenever its detection is imperfect.
// payload: The byte slice containing the image data to be converted.
func (c *Converter) FromPayload(payload []byte) (image.Image, error) {
	logrus.Debugf("converting image payload of size: %d bytes", len(payload))

	src, err := imaging.Decode(bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	if c.rotationDegrees != 0 {
		src = imaging.Rotate(src, c.rotationDegrees, color.Black)
	}

	smoothed := imaging.Blur(src, deblockSigma)
	img := stretchLevels(smoothed)

	if rect := detectDigitStripROI(src); rect != nil {
		logrus.Debugf("detected digit strip roi: %v", *rect)
		strip := imaging.Resize(imaging.Crop(smoothed, *rect), img.Bounds().Dx(), 0, imaging.CatmullRom)
		img = stack(img, stretchLevels(strip))
	} else {
		logrus.Debugf("no digit strip roi detected, using full image only")
	}

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

// stretchLevels linearly maps the image's low/high luminance percentiles to black/white, applying
// the same mapping to every channel so hues (e.g. the red decimal wheels) are preserved.
// img: The image to stretch.
func stretchLevels(img image.Image) image.Image {
	var hist [histogramBins]int
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			hist[luma(img.At(x, y))]++
		}
	}

	total := bounds.Dx() * bounds.Dy()
	lo, hi := percentile(hist, total, levelsLowPercentile), percentile(hist, total, levelsHighPercentile)
	if hi <= lo {
		return img
	}

	scale := float64(maxChannel) / float64(hi-lo)
	return imaging.AdjustFunc(img, func(c color.NRGBA) color.NRGBA {
		return color.NRGBA{
			R: stretchChannel(c.R, lo, scale),
			G: stretchChannel(c.G, lo, scale),
			B: stretchChannel(c.B, lo, scale),
			A: c.A,
		}
	})
}

// stretchChannel maps a channel value v to (v-lo)*scale, clamped to the 8-bit range.
// v: The channel value.
// lo: The value mapped to 0.
// scale: The multiplier applied after subtracting lo.
func stretchChannel(v uint8, lo int, scale float64) uint8 {
	return uint8(max(0, min(maxChannel, math.Round(float64(int(v)-lo)*scale))))
}

// percentile returns the smallest histogram bin at which the cumulative count exceeds fraction of total.
// hist: The luminance histogram.
// total: The total pixel count.
// fraction: The percentile, in [0, 1].
func percentile(hist [histogramBins]int, total int, fraction float64) int {
	target := int(float64(total) * fraction)
	cumulative := 0
	for i, count := range hist {
		cumulative += count
		if cumulative > target {
			return i
		}
	}
	return histogramBins - 1
}

// luma returns the 8-bit BT.601 luma of c.
// c: The color.
func luma(c color.Color) int {
	r, g, b, _ := c.RGBA()
	y := lumaR*float64(r>>rgba64To8BitShift) +
		lumaG*float64(g>>rgba64To8BitShift) +
		lumaB*float64(b>>rgba64To8BitShift)
	return min(maxChannel, int(math.Round(y)))
}

// stack places top above bottom in a single image of their combined height and the width of the wider one.
// top: The upper image.
// bottom: The lower image.
func stack(top, bottom image.Image) image.Image {
	out := imaging.New(
		max(top.Bounds().Dx(), bottom.Bounds().Dx()),
		top.Bounds().Dy()+bottom.Bounds().Dy(),
		color.Black,
	)
	out = imaging.Paste(out, top, image.Pt(0, 0))
	return imaging.Paste(out, bottom, image.Pt(0, top.Bounds().Dy()))
}

// Encode encodes an image.Image losslessly as PNG, so no second generation of JPEG compression
// artifacts is added on top of the camera's own.
// image: The image.Image object to be encoded.
func (c *Converter) Encode(image image.Image) ([]byte, error) {
	logrus.Debugf("encoding image: size=%dx%d", image.Bounds().Dx(), image.Bounds().Dy())

	var buf bytes.Buffer
	err := png.Encode(&buf, image)

	logrus.Debugf("encoded image: size=%d bytes", len(buf.Bytes()))

	return buf.Bytes(), err
}
