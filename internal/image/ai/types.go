package ai

import "context"

// ImageAI defines the interface for processing images using AI techniques.
type ImageAI interface {
	// HealthCheck verifies the overall health of the AI system, including connectivity and readiness.
	HealthCheck() bool
	// CheckModel verifies if the AI model is properly loaded and ready for processing.
	// ctx: The context for the operation, allowing for cancellation and timeouts.
	CheckModel(ctx context.Context) bool
	// ProcessImage takes raw image data as input and returns the watermeter data.
	// ctx: The context for the operation, allowing for cancellation and timeouts.
	// image: The raw image data to be processed.
	ProcessImage(ctx context.Context, image []byte) (*string, error)
}
