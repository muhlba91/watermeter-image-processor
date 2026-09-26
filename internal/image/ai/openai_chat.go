package ai

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/sirupsen/logrus"
)

// processImageOpenAICompat sends an image to an OpenAI-compatible chat completions endpoint.
// ctx: The context for the operation, allowing for cancellation and timeouts.
// client: An initialised openai.Client pointed at the target endpoint.
// model: The model identifier to request.
// image: The raw JPEG image data to be processed.
// providerName: A human-readable label used in log and error messages.
// temperature: The sampling temperature to request, or nil to leave it unset (the provider's
// default). Reasoning-tier models (e.g. OpenAI's o-series/gpt-5 family) reject any explicit value
// other than their default, so callers serving unknown or reasoning-capable backends should pass nil.
func processImageOpenAICompat(
	ctx context.Context,
	client *openai.Client,
	model string,
	image []byte,
	providerName string,
	temperature *float64,
) (*string, error) {
	imageURL := fmt.Sprintf("data:image/jpeg;base64,%s", base64.StdEncoding.EncodeToString(image))

	params := openai.ChatCompletionNewParams{
		Model:               model,
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
	if temperature != nil {
		params.Temperature = openai.Float(*temperature)
	}

	logrus.Debugf("processing image with %s model: %s", providerName, model)
	resp, err := client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from %s", providerName)
	}

	responseText := resp.Choices[0].Message.Content
	logrus.Debugf(
		"raw %s response: %q (finish_reason=%s)",
		providerName,
		responseText,
		resp.Choices[0].FinishReason,
	)

	result := cleanResult(responseText)
	return &result, nil
}
