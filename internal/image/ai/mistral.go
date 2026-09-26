package ai

import (
	"context"
	"errors"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/sirupsen/logrus"

	"github.com/muhlba91/watermeter-image-processor/cmd/configuration"
)

// mistralBaseURL is Mistral's OpenAI-compatible API base URL.
const mistralBaseURL = "https://api.mistral.ai/v1"

// Mistral is a struct that implements the ImageAI interface using Mistral's OpenAI-compatible chat completions API.
type Mistral struct {
	// client is the OpenAI-compatible API client pointed at Mistral's endpoint.
	client *openai.Client
	// model is the name of the Mistral model to be used for image processing.
	model string
}

// NewMistral creates a new instance of Mistral.
// configuration: The configuration data required to initialize the Mistral client.
func NewMistral(configuration *configuration.Data) (ImageAI, error) {
	client := openai.NewClient(
		option.WithBaseURL(mistralBaseURL),
		option.WithAPIKey(configuration.MistralAPIKey),
	)

	return &Mistral{
		client: &client,
		model:  configuration.MistralModel,
	}, nil
}

// HealthCheck verifies the overall health of the Mistral system by checking connectivity and responsiveness.
func (o *Mistral) HealthCheck() bool {
	ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
	defer cancel()

	_, err := o.client.Models.List(ctx)
	return err == nil
}

// CheckModel checks if the specified Mistral model exists and is available for processing images.
// ctx: The context for the operation, allowing for cancellation and timeouts.
func (o *Mistral) CheckModel(ctx context.Context) bool {
	requestedModel := o.model
	logrus.Debugf("checking mistral model '%s'", requestedModel)

	resp, err := o.client.Models.List(ctx)
	if err != nil {
		logrus.Errorf("error listing mistral models: %v", err)
		return false
	}

	for _, model := range resp.Data {
		logrus.Debugf("found mistral model: %s", model.ID)

		if model.ID == requestedModel {
			logrus.Debugf("mistral model '%s' is available", requestedModel)
			return true
		}
	}

	logrus.Warnf("mistral model '%s' is not available", requestedModel)
	return false
}

// ProcessImage processes the given image data using the Mistral library and returns the watermeter data.
// ctx: The context for the operation, allowing for cancellation and timeouts.
// image: The raw image data to be processed.
func (o *Mistral) ProcessImage(ctx context.Context, image []byte) (*string, error) {
	if !o.CheckModel(ctx) {
		return nil, errors.New("mistral model is not available")
	}

	// Mistral's chat models (unlike some reasoning-tier OpenAI models) reliably support a custom
	// temperature, so we pin it to 0 for deterministic, repeatable readings.
	temperature := 0.0
	return processImageOpenAICompat(ctx, o.client, o.model, image, ProviderMistral, &temperature)
}
