// Command eval measures how accurately and consistently the AI providers read the water meter test
// images in testing/images, and generates ESP32-like test images from a high-quality photo.
//
// Usage:
//
//	go run ./cmd/eval generate -src <photo.png> -accept <reading>[,<reading>] \
//		-framing <name>=x,y,w,h[@degrees]... -variant <framing>:<lit|dark>:<quality>... [-dark-mean <luminance>]
//	go run ./cmd/eval run [-images ...] [-providers ...] [-repeats 5] [-csv <file>] [-dump <dir>] [-v]
//
// The run command processes the images exactly like the processor and reads the AI provider API keys,
// models and IMAGE_ROTATION_DEGREES from the same environment variables.
package main

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

func main() {
	if len(os.Args) < 2 { //nolint:mnd // program name and command
		fmt.Fprintln(os.Stderr, "usage: eval <generate|run> [flags]")
		os.Exit(2) //nolint:mnd // usage error exit code
	}

	logrus.SetLevel(logrus.WarnLevel)

	var err error
	switch os.Args[1] {
	case "generate":
		err = generate(os.Args[2:])
	case "run":
		err = run(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
