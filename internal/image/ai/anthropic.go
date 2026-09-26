package ai

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/sirupsen/logrus"

	"github.com/muhlba91/watermeter-image-processor/cmd/configuration"
)

// anthropicTemperature defines the sampling temperature for the Anthropic response, kept at 0 for deterministic, repeatable readings.
const anthropicTemperature = 0.0

// Anthropic is a struct that implements the ImageAI interface using the Anthropic library.
type Anthropic struct {
	// client is the Anthropic API client used to process images.
	client *anthropic.Client
	// model is the name of the Anthropic model to be used for image processing.
	model string
}

// NewAnthropic creates a new instance of Anthropic.
// configuration: The configuration data required to initialize the Anthropic client.
func NewAnthropic(configuration *configuration.Data) (ImageAI, error) {
	client := anthropic.NewClient(
		option.WithAPIKey(configuration.AnthropicAPIKey),
	)

	return &Anthropic{
		client: &client,
		model:  configuration.AnthropicModel,
	}, nil
}

// HealthCheck verifies the overall health of the Anthropic system by checking connectivity and responsiveness.
func (o *Anthropic) HealthCheck() bool {
	ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
	defer cancel()

	_, err := o.client.Models.List(ctx, anthropic.ModelListParams{})
	return err == nil
}

// CheckModel checks if the specified Anthropic model exists and is available for processing images.
// ctx: The context for the operation, allowing for cancellation and timeouts.
func (o *Anthropic) CheckModel(ctx context.Context) bool {
	requestedModel := o.model
	logrus.Debugf("checking anthropic model '%s'", requestedModel)

	resp, err := o.client.Models.List(ctx, anthropic.ModelListParams{})
	if err != nil {
		logrus.Errorf("error listing anthropic models: %v", err)
		return false
	}

	for _, model := range resp.Data {
		logrus.Debugf("found anthropic model: %s", model.ID)

		if model.ID == requestedModel || strings.HasPrefix(model.ID, requestedModel+"-") {
			logrus.Debugf("anthropic model '%s' is available (matched: %s)", requestedModel, model.ID)
			return true
		}
	}

	logrus.Warnf("anthropic model '%s' is not available", requestedModel)
	return false
}

// ProcessImage processes the given image data using the Anthropic library and returns the watermeter data.
// ctx: The context for the operation, allowing for cancellation and timeouts.
// image: The raw image data to be processed.
func (o *Anthropic) ProcessImage(ctx context.Context, image []byte) (*string, error) {
	if !o.CheckModel(ctx) {
		return nil, errors.New("anthropic model is not available")
	}

	encodedImage := base64.StdEncoding.EncodeToString(image)

	logrus.Debugf("processing image with anthropic model: %s", o.model)
	resp, err := o.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:       o.model,
		MaxTokens:   maxResponseTokens,
		Temperature: anthropic.Float(anthropicTemperature),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewImageBlockBase64("image/jpeg", encodedImage),
				anthropic.NewTextBlock(getPrompt()),
			),
		},
	})
	if err != nil {
		return nil, err
	}

	if len(resp.Content) == 0 {
		return nil, errors.New("empty response from anthropic")
	}

	responseText := resp.Content[0].AsText().Text
	logrus.Debugf("raw anthropic response: %q", responseText)

	result := cleanResult(responseText)
	return &result, nil
}
