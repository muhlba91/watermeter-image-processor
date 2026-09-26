package ai

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/openai/openai-go"
	"github.com/sirupsen/logrus"
)

// openAICompatBase implements the shared logic for any provider backed by an OpenAI-compatible chat completions client.
// Providers embed this and only need to supply their own client construction.
type openAICompatBase struct {
	// client is the OpenAI-compatible API client pointed at the provider's endpoint.
	client *openai.Client
	// model is the name of the model to be requested.
	model string
	// providerName is a human-readable label used in log and error messages.
	providerName string
	// temperature is the sampling temperature to request, or nil to leave it unset (the provider's default).
	temperature *float64
	// modelCache caches the result of CheckModel for a TTL, avoiding a model list call on every image.
	modelCache *modelCache
}

// newOpenAICompatBase creates a new openAICompatBase.
// client: An initialised openai.Client pointed at the target endpoint.
// model: The model identifier to request.
// providerName: A human-readable label used in log and error messages.
// temperature: The sampling temperature to request, or nil to leave it unset.
// modelCheckCacheTTL: How long a CheckModel result is cached before being re-verified.
func newOpenAICompatBase(
	client *openai.Client,
	model string,
	providerName string,
	temperature *float64,
	modelCheckCacheTTL time.Duration,
) *openAICompatBase {
	return &openAICompatBase{
		client:       client,
		model:        model,
		providerName: providerName,
		temperature:  temperature,
		modelCache:   newModelCache(providerName, modelCheckCacheTTL),
	}
}

// HealthCheck verifies the overall health of the provider by checking connectivity and responsiveness.
func (o *openAICompatBase) HealthCheck() bool {
	ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
	defer cancel()

	_, err := o.client.Models.List(ctx)
	return err == nil
}

// CheckModel checks if the configured model exists and is available for processing images, caching the result for a TTL (see modelCache).
// ctx: The context for the operation, allowing for cancellation and timeouts.
func (o *openAICompatBase) CheckModel(ctx context.Context) bool {
	return o.modelCache.checkModelCached(func() bool {
		return o.checkModelUncached(ctx)
	})
}

// checkModelUncached performs the actual model-availability lookup.
// ctx: The context for the operation, allowing for cancellation and timeouts.
func (o *openAICompatBase) checkModelUncached(ctx context.Context) bool {
	logrus.Debugf("checking %s model '%s'", o.providerName, o.model)

	resp, err := o.client.Models.List(ctx)
	if err != nil {
		logrus.Errorf("error listing %s models: %v", o.providerName, err)
		return false
	}

	for _, model := range resp.Data {
		logrus.Debugf("found %s model: %s", o.providerName, model.ID)

		if model.ID == o.model {
			logrus.Debugf("%s model '%s' is available", o.providerName, o.model)
			return true
		}
	}

	logrus.Warnf("%s model '%s' is not available", o.providerName, o.model)
	return false
}

// ProcessImage sends the image to the provider's chat completions endpoint and returns the watermeter data.
// ctx: The context for the operation, allowing for cancellation and timeouts.
// image: The raw JPEG image data to be processed.
func (o *openAICompatBase) ProcessImage(ctx context.Context, image []byte) (*string, error) {
	if !o.CheckModel(ctx) {
		return nil, fmt.Errorf("%s model is not available", o.providerName)
	}

	imageURL := fmt.Sprintf("data:image/jpeg;base64,%s", base64.StdEncoding.EncodeToString(image))

	params := openai.ChatCompletionNewParams{
		Model:               o.model,
		MaxCompletionTokens: openai.Int(maxResponseTokens),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage([]openai.ChatCompletionContentPartUnionParam{
				openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
					URL:    imageURL,
					Detail: "high",
				}),
				openai.TextContentPart(getPrompt()),
			}),
		},
	}
	if o.temperature != nil {
		params.Temperature = openai.Float(*o.temperature)
	}

	logrus.Debugf("processing image with %s model: %s", o.providerName, o.model)
	resp, err := o.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from %s", o.providerName)
	}

	responseText := resp.Choices[0].Message.Content
	logrus.Debugf(
		"raw %s response: %q (finish_reason=%s)",
		o.providerName,
		responseText,
		resp.Choices[0].FinishReason,
	)

	result := cleanResult(responseText)
	return &result, nil
}
