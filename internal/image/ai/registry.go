package ai

import (
	"fmt"
	"sort"
	"strings"

	"github.com/muhlba91/watermeter-image-processor/cmd/configuration"
)

// ProviderGemini is the identifier for the Gemini AI provider.
const ProviderGemini = "gemini"

// ProviderOpenAI is the identifier for the OpenAI AI provider.
const ProviderOpenAI = "openai"

// ProviderAnthropic is the identifier for the Anthropic AI provider.
const ProviderAnthropic = "anthropic"

// ProviderOpenAICompat is the identifier for a generic OpenAI-compatible proxy provider (e.g. LiteLLM, Ollama, vLLM).
const ProviderOpenAICompat = "openai_compat"

// ProviderMistral is the identifier for the Mistral AI provider.
const ProviderMistral = "mistral"

// providerFactory is a function type that constructs an ImageAI from configuration.
type providerFactory func(*configuration.Data) (ImageAI, error)

// registeredProviders returns the registry of all supported AI providers.
func registeredProviders() map[string]providerFactory {
	return map[string]providerFactory{
		ProviderGemini:       NewGemini,
		ProviderOpenAI:       NewOpenAI,
		ProviderAnthropic:    NewAnthropic,
		ProviderOpenAICompat: NewOpenAICompat,
		ProviderMistral:      NewMistral,
	}
}

// NewProvider instantiates the ImageAI implementation selected by configuration.ModelProvider.
func NewProvider(cfg *configuration.Data) (ImageAI, error) {
	registry := registeredProviders()

	factory, ok := registry[cfg.ModelProvider]
	if !ok {
		return nil, fmt.Errorf(
			"unknown model provider %q: must be one of [%s]",
			cfg.ModelProvider,
			supportedProviders(registry),
		)
	}

	return factory(cfg)
}

// supportedProviders returns a sorted, comma-separated list of provider identifiers from the given registry.
func supportedProviders(registry map[string]providerFactory) string {
	keys := make([]string, 0, len(registry))
	for k := range registry {
		keys = append(keys, fmt.Sprintf("%q", k))
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}
