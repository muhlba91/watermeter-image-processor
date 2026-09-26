package ai

import (
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/muhlba91/watermeter-image-processor/cmd/configuration"
)

// OpenAI is a struct that implements the ImageAI interface using the OpenAI library.
type OpenAI struct {
	*openAICompatBase
}

// NewOpenAI creates a new instance of OpenAI.
// configuration: The configuration data required to initialize the OpenAI client.
func NewOpenAI(configuration *configuration.Data) (ImageAI, error) {
	client := openai.NewClient(
		option.WithAPIKey(configuration.OpenAIAPIKey),
	)

	// Temperature is deliberately left unset: reasoning-tier models (e.g. OpenAI's o-series/gpt-5
	// family) reject any explicit value other than their default (1).
	return &OpenAI{
		openAICompatBase: newOpenAICompatBase(
			&client,
			configuration.OpenAIModel,
			ProviderOpenAI,
			nil,
			configuration.ModelCheckCacheTTL,
		),
	}, nil
}
