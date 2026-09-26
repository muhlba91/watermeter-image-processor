package ai

import (
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/muhlba91/watermeter-image-processor/cmd/configuration"
)

// OpenAICompat is a struct that implements the ImageAI interface using any OpenAI-compatible proxy
// (e.g. LiteLLM, Ollama, vLLM). The target URL and an optional API key are read from configuration.
type OpenAICompat struct {
	*openAICompatBase
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

	// Temperature is deliberately left unset: this provider serves arbitrary OpenAI-compatible
	// proxies whose model capabilities we cannot know in advance.
	return &OpenAICompat{
		openAICompatBase: newOpenAICompatBase(
			&client,
			configuration.OpenAICompatModel,
			ProviderOpenAICompat,
			nil,
			configuration.ModelCheckCacheTTL,
		),
	}, nil
}
