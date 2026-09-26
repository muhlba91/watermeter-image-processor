package ai

import (
	"context"
	"errors"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/genai"

	"github.com/muhlba91/watermeter-image-processor/cmd/configuration"
)

// healthCheckTimeout defines the timeout duration for health checks related to the S3 client.
const healthCheckTimeout = 2 * time.Second

// listPageSize defines the number of models to retrieve per page when listing Gemini models.
const listPageSize = 100

// Gemini is a struct that implements the ImageAI interface using the Gemini library.
type Gemini struct {
	// client is the Gemini API client used to process images.
	client *genai.Client
	// model is the name of the Gemini model to be used for image processing.
	model string
}

// NewGemini creates a new instance of Gemini.
// configuration: The configuration data required to initialize the Gemini client.
func NewGemini(configuration *configuration.Data) (ImageAI, error) {
	config := &genai.ClientConfig{
		APIKey:  configuration.GeminiAPIKey,
		Backend: genai.BackendGeminiAPI,
	}
	client, cErr := genai.NewClient(context.Background(), config)
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

	_, err := o.client.Models.List(ctx, &genai.ListModelsConfig{})
	return err == nil
}

// CheckModel checks if the specified Gemini model exists and is available for processing images.
// ctx: The context for the operation, allowing for cancellation and timeouts.
func (o *Gemini) CheckModel(ctx context.Context) bool {
	requestedModel := o.model
	logrus.Debugf("checking gemini model '%s'", requestedModel)

	resp, err := o.client.Models.List(ctx, &genai.ListModelsConfig{
		PageSize: listPageSize,
	})
	if err != nil {
		logrus.Errorf("error listing gemini models: %v", err)
		return false
	}

	for _, model := range resp.Items {
		logrus.Debugf("found gemini model: %s", model.Name)

		if model.Name == requestedModel || model.Name == "models/"+requestedModel {
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

	temperature := float32(0.0)
	config := &genai.GenerateContentConfig{
		Temperature:     &temperature,
		MaxOutputTokens: maxResponseTokens,
	}

	contents := []*genai.Content{
		{
			Role: "user",
			Parts: []*genai.Part{
				{
					InlineData: &genai.Blob{
						MIMEType: "image/jpeg",
						Data:     image,
					},
				},
				{
					Text: getPrompt(),
				},
			},
		},
	}

	logrus.Debugf("processing image with gemini model: %s", o.model)
	resp, err := o.client.Models.GenerateContent(ctx, o.model, contents, config)
	if err != nil {
		return nil, err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("empty response from gemini")
	}

	responseText := resp.Candidates[0].Content.Parts[0].Text
	logrus.Debugf("raw gemini response: %q", responseText)

	result := cleanResult(responseText)
	return &result, nil
}
