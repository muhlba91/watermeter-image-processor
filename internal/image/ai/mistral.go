package ai

import (
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/muhlba91/watermeter-image-processor/cmd/configuration"
)

// mistralBaseURL is Mistral's OpenAI-compatible API base URL.
const mistralBaseURL = "https://api.mistral.ai/v1"

// mistralTemperature defines the sampling temperature for the Mistral response. Unlike some
// reasoning-tier OpenAI models, Mistral's chat models reliably support a custom temperature, so it
// is pinned to 0 for deterministic, repeatable readings.
const mistralTemperature = 0.0

// Mistral is a struct that implements the ImageAI interface using Mistral's OpenAI-compatible chat completions API.
type Mistral struct {
	*openAICompatBase
}

// NewMistral creates a new instance of Mistral.
// configuration: The configuration data required to initialize the Mistral client.
func NewMistral(configuration *configuration.Data) (ImageAI, error) {
	client := openai.NewClient(
		option.WithBaseURL(mistralBaseURL),
		option.WithAPIKey(configuration.MistralAPIKey),
	)

	temperature := mistralTemperature
	return &Mistral{
		openAICompatBase: newOpenAICompatBase(
			&client,
			configuration.MistralModel,
			ProviderMistral,
			&temperature,
			configuration.ModelCheckCacheTTL,
		),
	}, nil
}
