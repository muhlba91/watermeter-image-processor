package ai

import (
	"context"
	"errors"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/sirupsen/logrus"

	"github.com/muhlba91/watermeter-image-processor/cmd/configuration"
)

// OpenAICompat is a struct that implements the ImageAI interface using any OpenAI-compatible proxy
// (e.g. LiteLLM, Ollama, vLLM). The target URL and an optional API key are read from configuration.
type OpenAICompat struct {
	// client is the OpenAI-compatible API client pointed at the proxy endpoint.
	client *openai.Client
	// model is the name of the model to be requested from the proxy.
	model string
	// modelCache caches the result of CheckModel for a TTL, avoiding a model list call on every image.
	modelCache *modelCache
}

// NewOpenAICompat creates a new instance of OpenAICompat.
// configuration: The configuration data required to initialize the OpenAI-compatible client.
func NewOpenAICompat(configuration *configuration.Data) (ImageAI, error) {
	opts := []option.RequestOption{
		option.WithBaseURL(configuration.OpenAICompatURL),
	}
	if configuration.OpenAICompatAPIKey != "" {
		opts = append(opts, option.WithAPIKey(configuration.OpenAICompatAPIKey))
	}

	client := openai.NewClient(opts...)

	return &OpenAICompat{
		client:     &client,
		model:      configuration.OpenAICompatModel,
		modelCache: newModelCache(ProviderOpenAICompat, configuration.ModelCheckCacheTTL),
	}, nil
}

// HealthCheck verifies the overall health of the OpenAI-compatible proxy by checking connectivity and responsiveness.
func (o *OpenAICompat) HealthCheck() bool {
	ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
	defer cancel()

	_, err := o.client.Models.List(ctx)
	return err == nil
}

// CheckModel checks if the specified model is available via the OpenAI-compatible proxy.
// ctx: The context for the operation, allowing for cancellation and timeouts.
func (o *OpenAICompat) CheckModel(ctx context.Context) bool {
	return o.modelCache.checkModelCached(func() bool {
		return o.checkModelUncached(ctx)
	})
}

// checkModelUncached performs the actual OpenAI-compatible model-availability lookup.
// ctx: The context for the operation, allowing for cancellation and timeouts.
func (o *OpenAICompat) checkModelUncached(ctx context.Context) bool {
	requestedModel := o.model
	logrus.Debugf("checking openai_compat model '%s'", requestedModel)

	resp, err := o.client.Models.List(ctx)
	if err != nil {
		logrus.Errorf("error listing openai_compat models: %v", err)
		return false
	}

	for _, model := range resp.Data {
		logrus.Debugf("found openai_compat model: %s", model.ID)

		if model.ID == requestedModel {
			logrus.Debugf("openai_compat model '%s' is available", requestedModel)
			return true
		}
	}

	logrus.Warnf("openai_compat model '%s' is not available", requestedModel)
	return false
}

// ProcessImage processes the given image data via the OpenAI-compatible proxy and returns the watermeter data.
// ctx: The context for the operation, allowing for cancellation and timeouts.
// image: The raw image data to be processed.
func (o *OpenAICompat) ProcessImage(ctx context.Context, image []byte) (*string, error) {
	if !o.CheckModel(ctx) {
		return nil, errors.New("openai_compat model is not available")
	}

	return processImageOpenAICompat(ctx, o.client, o.model, image, ProviderOpenAICompat, nil)
}
