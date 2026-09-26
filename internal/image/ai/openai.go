package ai

import (
	"context"
	"errors"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/sirupsen/logrus"

	"github.com/muhlba91/watermeter-image-processor/cmd/configuration"
)

// OpenAI is a struct that implements the ImageAI interface using the OpenAI library.
type OpenAI struct {
	// client is the OpenAI API client used to process images.
	client *openai.Client
	// model is the name of the OpenAI model to be used for image processing.
	model string
	// modelCache caches the result of CheckModel for a TTL, avoiding a model list call on every image.
	modelCache *modelCache
}

// NewOpenAI creates a new instance of OpenAI.
// configuration: The configuration data required to initialize the OpenAI client.
func NewOpenAI(configuration *configuration.Data) (ImageAI, error) {
	client := openai.NewClient(
		option.WithAPIKey(configuration.OpenAIAPIKey),
	)

	return &OpenAI{
		client:     &client,
		model:      configuration.OpenAIModel,
		modelCache: newModelCache(ProviderOpenAI, configuration.ModelCheckCacheTTL),
	}, nil
}

// HealthCheck verifies the overall health of the OpenAI system by checking connectivity and responsiveness.
func (o *OpenAI) HealthCheck() bool {
	ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
	defer cancel()

	_, err := o.client.Models.List(ctx)
	return err == nil
}

// CheckModel checks if the specified OpenAI model exists and is available for processing images.
// ctx: The context for the operation, allowing for cancellation and timeouts.
func (o *OpenAI) CheckModel(ctx context.Context) bool {
	return o.modelCache.checkModelCached(func() bool {
		return o.checkModelUncached(ctx)
	})
}

// checkModelUncached performs the actual OpenAI model-availability lookup.
// ctx: The context for the operation, allowing for cancellation and timeouts.
func (o *OpenAI) checkModelUncached(ctx context.Context) bool {
	requestedModel := o.model
	logrus.Debugf("checking openai model '%s'", requestedModel)

	resp, err := o.client.Models.List(ctx)
	if err != nil {
		logrus.Errorf("error listing openai models: %v", err)
		return false
	}

	for _, model := range resp.Data {
		logrus.Debugf("found openai model: %s", model.ID)

		if model.ID == requestedModel {
			logrus.Debugf("openai model '%s' is available", requestedModel)
			return true
		}
	}

	logrus.Warnf("openai model '%s' is not available", requestedModel)
	return false
}

// ProcessImage processes the given image data using the OpenAI library and returns the watermeter data.
// ctx: The context for the operation, allowing for cancellation and timeouts.
// image: The raw image data to be processed.
func (o *OpenAI) ProcessImage(ctx context.Context, image []byte) (*string, error) {
	if !o.CheckModel(ctx) {
		return nil, errors.New("openai model is not available")
	}

	return processImageOpenAICompat(ctx, o.client, o.model, image, ProviderOpenAI, nil)
}
