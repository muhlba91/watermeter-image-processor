package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
)

const (
	// esp32Width and esp32Height are the ESP32-CAM's configured SVGA resolution.
	esp32Width  = 800
	esp32Height = 600
	// maxChannel is the maximum 8-bit channel value.
	maxChannel = 255
	// rectParts is the number of comma-separated values in a framing's crop rectangle.
	rectParts = 4
	// variantParts is the number of colon-separated values in a variant.
	variantParts = 3
	// lumaR, lumaG and lumaB are the ITU-R BT.601 luma weights.
	lumaR = 0.299
	lumaG = 0.587
	lumaB = 0.114
	// rgba64To8Bit converts a 16-bit color channel to the 8-bit range.
	rgba64To8Bit = 257
	// halfTurnDegrees is the number of degrees in pi radians.
	halfTurnDegrees = 180
	// defaultDarkMean is the mean luminance of the under-exposed ("dark") variants, matching
	// testing/watermeter.jpg.
	defaultDarkMean = 28

	// lightingLit keeps the source photo's exposure.
	lightingLit = "lit"
	// lightingDark scales the photo down to the -dark-mean luminance, simulating an under-exposed frame.
	lightingDark = "dark"
)

// framing is a crop of the source photo, optionally rotated to level the digit wheels.
type framing struct {
	// rect is the source crop.
	rect image.Rectangle
	// rotation is the counter-clockwise rotation, in degrees, applied after cropping.
	rotation float64
}

// framingFlags collects repeated -framing flags by name.
type framingFlags map[string]framing

// String implements flag.Value.
func (f framingFlags) String() string {
	return fmt.Sprint(map[string]framing(f))
}

// Set parses a "name=x,y,w,h[@degrees]" framing.
// value: The flag value.
func (f framingFlags) Set(value string) error {
	name, spec, ok := strings.Cut(value, "=")
	if !ok || name == "" {
		return fmt.Errorf("invalid framing %q: expected name=x,y,w,h[@degrees]", value)
	}

	rectSpec, rotationSpec, rotated := strings.Cut(spec, "@")
	rect, err := parseRect(rectSpec)
	if err != nil {
		return err
	}

	fr := framing{rect: rect}
	if rotated {
		if fr.rotation, err = strconv.ParseFloat(rotationSpec, 64); err != nil {
			return fmt.Errorf("invalid framing rotation %q: %w", rotationSpec, err)
		}
	}
	f[name] = fr
	return nil
}

// variant is one generated test image: a framing, a lighting and a JPEG quality.
type variant struct {
	// framing is the name of the framing.
	framing string
	// lighting is lightingLit or lightingDark.
	lighting string
	// quality is the JPEG quality (1-100).
	quality int
}

// file returns the variant's file name, e.g. "level-dark-q10.jpg".
func (v variant) file() string {
	return fmt.Sprintf("%s-%s-q%d.jpg", v.framing, v.lighting, v.quality)
}

// variantFlags collects repeated -variant flags.
type variantFlags []variant

// String implements flag.Value.
func (v *variantFlags) String() string {
	return fmt.Sprint(*v)
}

// Set parses a "framing:lighting:quality" variant.
// value: The flag value.
func (v *variantFlags) Set(value string) error {
	parts := strings.Split(value, ":")
	if len(parts) != variantParts {
		return fmt.Errorf("invalid variant %q: expected framing:lighting:quality", value)
	}
	if parts[1] != lightingLit && parts[1] != lightingDark {
		return fmt.Errorf("invalid variant %q: lighting must be %q or %q", value, lightingLit, lightingDark)
	}

	quality, err := strconv.Atoi(parts[2])
	if err != nil {
		return fmt.Errorf("invalid variant %q: %w", value, err)
	}
	*v = append(*v, variant{framing: parts[0], lighting: parts[1], quality: quality})
	return nil
}

// generate writes ESP32-like variants of a high-quality source photo into the image directory and
// registers them in its manifest.
// args: The command-line flags.
func generate(args []string) error {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	src := fs.String("src", "", "high-quality source photo (PNG/JPEG; convert a DNG first, e.g. with sips)")
	accept := fs.String("accept", "", "comma-separated readings accepted as correct")
	darkMean := fs.Float64("dark-mean", defaultDarkMean, "mean luminance (0-255) of the dark variants")
	dir := fs.String("dir", "testing/images", "image directory")
	framings := framingFlags{}
	fs.Var(framings, "framing", "name=x,y,w,h[@degrees] crop of the source photo, optionally rotated "+
		"counter-clockwise (repeatable)")
	var variants variantFlags
	fs.Var(&variants, "variant", "framing:lighting:quality image to generate, lighting being lit or dark (repeatable)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *src == "" || *accept == "" || len(variants) == 0 {
		return errors.New("-src, -accept and at least one -variant are required")
	}

	source, err := imaging.Open(*src)
	if err != nil {
		return err
	}

	m, err := loadManifest(*dir)
	if err != nil {
		return err
	}

	for _, v := range variants {
		fr, ok := framings[v.framing]
		if !ok {
			return fmt.Errorf("variant %q uses undefined framing %q", v.file(), v.framing)
		}

		img := frame(source, fr)
		if v.lighting == lightingDark {
			img = scaleToMean(img, *darkMean)
		}

		if wErr := writeJPEG(filepath.Join(*dir, v.file()), img, v.quality); wErr != nil {
			return wErr
		}
		m.upsert(testImage{
			File:   v.file(),
			Accept: strings.Split(*accept, ","),
			Note: fmt.Sprintf("generated from %s: %s framing, %s lighting, JPEG quality %d",
				filepath.Base(*src), v.framing, v.lighting, v.quality),
		})
		fmt.Fprintln(os.Stdout, "wrote", v.file())
	}

	return m.save(*dir)
}

// frame crops and rotates the source photo per the framing, and fills the ESP32 resolution. A
// rotated crop is trimmed to the largest centered 4:3 rectangle without empty corners.
// source: The source photo.
// fr: The framing.
func frame(source image.Image, fr framing) image.Image {
	img := imaging.Crop(source, fr.rect)

	if fr.rotation != 0 {
		w, h := float64(img.Bounds().Dx()), float64(img.Bounds().Dy())
		radians := fr.rotation * math.Pi / halfTurnDegrees
		sin, cos := math.Abs(math.Sin(radians)), math.Abs(math.Cos(radians))
		aspect := float64(esp32Height) / float64(esp32Width)
		// an axis-aligned cw x ch rectangle fits inside the rotated w x h crop when
		// cw*cos + ch*sin <= w and cw*sin + ch*cos <= h
		cw := min(w/(cos+aspect*sin), h/(sin+aspect*cos))
		img = imaging.CropCenter(imaging.Rotate(img, fr.rotation, color.Black), int(cw), int(cw*aspect))
	}

	return imaging.Fill(img, esp32Width, esp32Height, imaging.Center, imaging.Lanczos)
}

// parseRect parses an "x,y,w,h" rectangle.
// s: The rectangle string.
func parseRect(s string) (image.Rectangle, error) {
	parts := strings.Split(s, ",")
	if len(parts) != rectParts {
		return image.Rectangle{}, fmt.Errorf("invalid rectangle %q: expected x,y,w,h", s)
	}

	var v [rectParts]int
	for i, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return image.Rectangle{}, fmt.Errorf("invalid rectangle %q: %w", s, err)
		}
		v[i] = n
	}
	return image.Rect(v[0], v[1], v[0]+v[2], v[1]+v[3]), nil
}

// scaleToMean darkens or brightens img linearly so its mean luminance equals target, simulating
// an under-exposed frame.
// img: The image to scale.
// target: The desired mean luminance (0-255).
func scaleToMean(img image.Image, target float64) image.Image {
	bounds := img.Bounds()
	var sum float64
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			sum += (lumaR*float64(r) + lumaG*float64(g) + lumaB*float64(b)) / rgba64To8Bit
		}
	}
	factor := target / (sum / float64(bounds.Dx()*bounds.Dy()))

	scale := func(v uint8) uint8 { return uint8(min(maxChannel, math.Round(float64(v)*factor))) }
	return imaging.AdjustFunc(img, func(c color.NRGBA) color.NRGBA {
		return color.NRGBA{R: scale(c.R), G: scale(c.G), B: scale(c.B), A: c.A}
	})
}

// writeJPEG encodes img as a 4:2:0 chroma-subsampled JPEG, like the ESP32-CAM, at the given quality.
// path: The output file path.
// img: The image to encode.
// quality: The JPEG quality (1-100).
func writeJPEG(path string, img image.Image, quality int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return jpeg.Encode(f, img, &jpeg.Options{Quality: quality})
}
