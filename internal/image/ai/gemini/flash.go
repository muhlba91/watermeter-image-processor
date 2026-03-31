package gemini

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/muhlba91/watermeter-image-processor/cmd/configuration"
	"github.com/muhlba91/watermeter-image-processor/internal/image/ai"
)

// healthCheckTimeout defines the timeout duration for health checks related to the S3 client.
const healthCheckTimeout = 2 * time.Second

// expectedDigits defines the number of digits expected in the water meter reading.
const expectedDigits = 5

// Gemini is a struct that implements the ImageAI interface using the Gemini library.
type Gemini struct {
	// client is the Gemini API client used to process images.
	client *genai.Client
	// model is the name of the Gemini model to be used for image processing.
	model string
}

// NewGemini creates a new instance of Gemini.
// configuration: The configuration data required to initialize the Gemini client.
func NewGemini(configuration *configuration.Data) (ai.ImageAI, error) {
	client, cErr := genai.NewClient(context.Background(), option.WithAPIKey(configuration.GeminiAPIKey))
	if cErr != nil {
		return nil, cErr
	}

	return &Gemini{
		client: client,
		model:  configuration.GeminiModel,
	}, nil
}

// HealthCheck verifies the overall health of the Gemini system by checking connectivity and responsiveness.
func (o *Gemini) HealthCheck() bool {
	ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
	defer cancel()

	iter := o.client.ListModels(ctx)
	_, err := iter.Next()
	return err == nil
}

// CheckModel checks if the specified Gemini model exists and is available for processing images.
// ctx: The context for the operation, allowing for cancellation and timeouts.
func (o *Gemini) CheckModel(ctx context.Context) bool {
	requestedModel := fmt.Sprintf("models/%s", o.model)
	logrus.Debugf("checking gemini model '%s'", requestedModel)

	iter := o.client.ListModels(ctx)
	for {
		model, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			logrus.Errorf("error listing gemini models: %v", err)
			return false
		}

		logrus.Debugf("found gemini model: %s", model.Name)

		if model.Name == requestedModel {
			logrus.Debugf("gemini model '%s' is available", requestedModel)
			return true
		}
	}

	logrus.Warnf("gemini model '%s' is not available", requestedModel)
	return false
}

// ProcessImage processes the given image data using the Gemini library and returns the watermeter data.
// ctx: The context for the operation, allowing for cancellation and timeouts.
// image: The raw image data to be processed.
func (o *Gemini) ProcessImage(ctx context.Context, image []byte) (*string, error) {
	if !o.CheckModel(ctx) {
		return nil, errors.New("gemini model is not available")
	}

	model := o.client.GenerativeModel(o.model)
	model.SetTemperature(0.0)

	prompt := []genai.Part{
		genai.ImageData("jpeg", image),
		genai.Text("Read the 5-digit number on the water meter. Output ONLY the digits."),
	}

	logrus.Debugf("processing image with gemini model: %s", o.model)
	resp, err := model.GenerateContent(ctx, prompt...)
	if err != nil {
		return nil, err
	}

	result := o.cleanResult(fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]))
	return &result, nil
}

// cleanResult takes the raw result from the Gemini model and extracts the relevant digits, ensuring that the output is in the expected format for water meter readings.
// result: The raw result string obtained from the Gemini model, which may contain extraneous characters and formatting.
func (o *Gemini) cleanResult(result string) string {
	re := regexp.MustCompile(`[^0-9]`)
	digits := re.ReplaceAllString(result, "")

	if len(digits) != expectedDigits {
		logrus.Errorf("unexpected result length: got %d digits, expected 5. result: '%s'", len(digits), result)
		return digits
	}

	formatted := fmt.Sprintf("%s.%s", digits[:3], digits[3:])
	val, err := strconv.ParseFloat(formatted, 64)
	if err != nil {
		logrus.Warnf("error parsing float from cleaned result: %v", err)
		return formatted
	}

	return strconv.FormatFloat(val, 'f', -1, 64)
}
