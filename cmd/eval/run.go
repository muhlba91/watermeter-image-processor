package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/muhlba91/watermeter-image-processor/cmd/configuration"
	internalimg "github.com/muhlba91/watermeter-image-processor/internal/image"
	"github.com/muhlba91/watermeter-image-processor/internal/image/ai"
)

const (
	// requestTimeout bounds a single provider call.
	requestTimeout = 90 * time.Second
	// readingTolerance is the maximum numeric difference for a reading to match an accepted one.
	readingTolerance = 1e-6
	// percent converts a fraction to a percentage.
	percent = 100
	// defaultRepeats is the default number of calls per image and provider.
	defaultRepeats = 5
	// defaultParallel is the default number of concurrent calls per provider.
	defaultParallel = 3
	// columnPadding is the padding between summary table columns.
	columnPadding = 2
)

// cell identifies one configuration in the evaluation matrix.
type cell struct {
	// image is the test image file.
	image string
	// provider is the AI provider name.
	provider string
}

// outcome is the result of a single provider call.
type outcome struct {
	cell

	// repeat is the repeat index of the call.
	repeat int
	// reading is the published reading, empty if the call failed.
	reading string
	// correct reports whether reading is one of the accepted readings.
	correct bool
	// err is the provider error, including rejected answers.
	err error
	// latency is the duration of the call.
	latency time.Duration
}

// preparedImage is a test image processed by the converter, ready to send.
type preparedImage struct {
	// file is the test image file.
	file string
	// data is the processed image.
	data []byte
}

// run sends every selected test image to every selected provider, repeating each call to measure
// consistency, and prints a summary table.
// args: The command-line flags.
func run(args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	dir := fs.String("dir", "testing/images", "image directory containing manifest.json")
	images := fs.String("images", "", "comma-separated substrings; only images whose file contains one are evaluated")
	providers := fs.String("providers", "gemini,openai,anthropic,mistral", "comma-separated providers")
	repeats := fs.Int("repeats", defaultRepeats, "calls per image and provider")
	parallel := fs.Int("parallel", defaultParallel, "concurrent calls per provider")
	out := fs.String("csv", "", "optional CSV file for the raw results")
	dump := fs.String("dump", "", "optional directory to write the processed images to")
	verbose := fs.Bool("v", false, "log the raw provider responses")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}

	m, err := loadManifest(*dir)
	if err != nil {
		return err
	}
	selected := filterImages(m.Images, *images)
	if len(selected) == 0 {
		return fmt.Errorf("no test images selected in %s", *dir)
	}

	cfg := configuration.Init()
	prepared, err := prepareImages(*dir, selected, internalimg.NewConverter(cfg.ImageRotationDegrees), *dump)
	if err != nil {
		return err
	}

	outcomes, err := callProviders(&cfg, selected, prepared, split(*providers), *repeats, *parallel)
	if err != nil {
		return err
	}

	if *out != "" {
		if wErr := writeCSV(*out, outcomes); wErr != nil {
			return wErr
		}
	}
	printSummary(outcomes)
	return nil
}

// filterImages returns the images whose file name contains any of the comma-separated substrings,
// or all images if filter is empty.
// all: The manifest's images.
// filter: The comma-separated substrings.
func filterImages(all []testImage, filter string) []testImage {
	if filter == "" {
		return all
	}

	var selected []testImage
	for _, img := range all {
		for _, f := range split(filter) {
			if strings.Contains(img.File, f) {
				selected = append(selected, img)
				break
			}
		}
	}
	return selected
}

// prepareImages runs the processor's conversion pipeline on every selected image.
// dir: The image directory.
// images: The selected test images.
// converter: The converter, configured like the processor.
// dump: An optional directory to write the processed images to.
func prepareImages(
	dir string,
	images []testImage,
	converter *internalimg.Converter,
	dump string,
) ([]preparedImage, error) {
	prepared := make([]preparedImage, 0, len(images))
	for _, img := range images {
		raw, err := os.ReadFile(filepath.Join(dir, img.File))
		if err != nil {
			return nil, err
		}

		converted, cErr := converter.FromPayload(raw)
		if cErr != nil {
			return nil, fmt.Errorf("%s: %w", img.File, cErr)
		}
		data, eErr := converter.Encode(converted)
		if eErr != nil {
			return nil, fmt.Errorf("%s: %w", img.File, eErr)
		}
		prepared = append(prepared, preparedImage{file: img.File, data: data})

		if dump != "" {
			if mErr := os.MkdirAll(dump, 0o750); mErr != nil {
				return nil, mErr
			}
			name := strings.TrimSuffix(filepath.Base(img.File), filepath.Ext(img.File)) + ".png"
			//nolint:gosec // the dump directory is an operator-supplied command-line flag
			if wErr := os.WriteFile(filepath.Join(dump, name), data, 0o600); wErr != nil {
				return nil, wErr
			}
		}
	}
	return prepared, nil
}

// callProviders sends every prepared image to every provider repeats times, with up to parallel
// concurrent calls per provider.
// cfg: The processor configuration, for the provider API keys and models.
// images: The selected test images, for the accepted readings.
// prepared: The processed images.
// providers: The provider names.
// repeats: The number of calls per image and provider.
// parallel: The maximum concurrent calls per provider.
func callProviders(
	cfg *configuration.Data,
	images []testImage,
	prepared []preparedImage,
	providers []string,
	repeats, parallel int,
) ([]outcome, error) {
	accepted := map[string][]string{}
	for _, img := range images {
		accepted[img.File] = img.Accept
	}

	var (
		mu       sync.Mutex
		outcomes []outcome
		wg       sync.WaitGroup
	)

	for _, name := range providers {
		providerCfg := *cfg
		providerCfg.ModelProvider = name
		provider, err := ai.NewProvider(&providerCfg)
		if err != nil {
			return nil, err
		}

		sem := make(chan struct{}, parallel)
		for _, p := range prepared {
			for i := range repeats {
				wg.Go(func() {
					sem <- struct{}{}
					defer func() { <-sem }()

					o := callOnce(provider, p, i)
					o.provider = name
					o.correct = o.err == nil && matches(o.reading, accepted[p.file])

					mu.Lock()
					outcomes = append(outcomes, o)
					fmt.Fprintf(os.Stderr, "\r%d calls done", len(outcomes))
					mu.Unlock()
				})
			}
		}
	}

	wg.Wait()
	fmt.Fprintln(os.Stderr)
	return outcomes, nil
}

// callOnce sends one prepared image to the provider.
// provider: The AI provider.
// p: The processed image.
// repeat: The repeat index, for reporting.
func callOnce(provider ai.ImageAI, p preparedImage, repeat int) outcome {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	start := time.Now()
	reading, err := provider.ProcessImage(ctx, p.data)
	o := outcome{
		cell:    cell{image: p.file},
		repeat:  repeat,
		err:     err,
		latency: time.Since(start),
	}
	if reading != nil {
		o.reading = *reading
	}
	return o
}

// matches reports whether reading numerically equals one of the accepted readings.
// reading: The provider's reading.
// accept: The accepted readings.
func matches(reading string, accept []string) bool {
	got, err := strconv.ParseFloat(reading, 64)
	if err != nil {
		return false
	}
	for _, a := range accept {
		want, aErr := strconv.ParseFloat(a, 64)
		if aErr == nil && math.Abs(got-want) < readingTolerance {
			return true
		}
	}
	return false
}

// writeCSV writes the raw outcomes to path.
// path: The CSV file path.
// outcomes: The outcomes.
func writeCSV(path string, outcomes []outcome) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	_ = w.Write([]string{"image", "provider", "repeat", "reading", "correct", "error", "latency_ms"})
	for _, o := range outcomes {
		errText := ""
		if o.err != nil {
			errText = o.err.Error()
		}
		_ = w.Write([]string{
			o.image, o.provider, strconv.Itoa(o.repeat), o.reading, strconv.FormatBool(o.correct), errText,
			strconv.FormatInt(o.latency.Milliseconds(), 10),
		})
	}
	w.Flush()
	return w.Error()
}

// printSummary prints, per image and provider, the correct-reading rate, the share of the most
// common reading (consistency) and the distinct readings seen, followed by per-provider totals.
// outcomes: The outcomes.
func printSummary(outcomes []outcome) {
	byCell := map[cell][]outcome{}
	byProvider := map[string][]outcome{}
	for _, o := range outcomes {
		byCell[o.cell] = append(byCell[o.cell], o)
		byProvider[o.provider] = append(byProvider[o.provider], o)
	}

	cells := make([]cell, 0, len(byCell))
	for c := range byCell {
		cells = append(cells, c)
	}
	sort.Slice(cells, func(i, j int) bool {
		if cells[i].image != cells[j].image {
			return cells[i].image < cells[j].image
		}
		return cells[i].provider < cells[j].provider
	})

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, columnPadding, ' ', 0)
	fmt.Fprintln(tw, "IMAGE\tPROVIDER\tCORRECT\tMODAL\tREADINGS")
	for _, c := range cells {
		correct, modal, readings := stats(byCell[c])
		n := len(byCell[c])
		fmt.Fprintf(tw, "%s\t%s\t%d/%d\t%d/%d\t%s\n", c.image, c.provider, correct, n, modal, n, readings)
	}
	_ = tw.Flush()

	providers := make([]string, 0, len(byProvider))
	for p := range byProvider {
		providers = append(providers, p)
	}
	sort.Strings(providers)

	fmt.Fprintln(os.Stdout)
	tw = tabwriter.NewWriter(os.Stdout, 0, 0, columnPadding, ' ', 0)
	fmt.Fprintln(tw, "PROVIDER\tCORRECT\tACCURACY")
	for _, p := range providers {
		correct, _, _ := stats(byProvider[p])
		n := len(byProvider[p])
		fmt.Fprintf(tw, "%s\t%d/%d\t%.0f%%\n", p, correct, n, percent*float64(correct)/float64(n))
	}
	_ = tw.Flush()
}

// stats returns the number of correct outcomes, the count of the most common reading, and a
// "reading×count" list of the distinct readings (failed or rejected calls shown as ERR).
// outcomes: The outcomes of one configuration.
func stats(outcomes []outcome) (int, int, string) {
	correct := 0
	counts := map[string]int{}
	for _, o := range outcomes {
		if o.correct {
			correct++
		}
		r := o.reading
		if o.err != nil {
			r = "ERR"
		}
		counts[r]++
	}

	distinct := make([]string, 0, len(counts))
	modal := 0
	for r, n := range counts {
		distinct = append(distinct, r)
		modal = max(modal, n)
	}
	sort.Slice(distinct, func(i, j int) bool { return counts[distinct[i]] > counts[distinct[j]] })

	parts := make([]string, len(distinct))
	for i, r := range distinct {
		parts[i] = fmt.Sprintf("%s×%d", r, counts[r])
	}
	return correct, modal, strings.Join(parts, " ")
}

// split splits a comma-separated list, dropping empty items.
// s: The comma-separated list.
func split(s string) []string {
	var out []string
	for p := range strings.SplitSeq(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
