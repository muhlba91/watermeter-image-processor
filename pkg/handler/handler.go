package handler

import (
	"context"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"

	internalimg "github.com/muhlba91/watermeter-image-processor/internal/image"
	"github.com/muhlba91/watermeter-image-processor/internal/image/ai"
	"github.com/muhlba91/watermeter-image-processor/internal/mqtt/publisher"
	"github.com/muhlba91/watermeter-image-processor/internal/scaleway/object"
)

// Handler is responsible for processing incoming MQTT messages.
type Handler struct {
	converter *internalimg.Converter
	ai        ai.ImageAI
	publisher *publisher.Publisher
	uploader  *object.Uploader
}

// NewHandler creates a new instance of the Handler.
// publisher: The MQTT publisher to publish results.
// uploader: The object uploader to upload images.
// aiProvider: The AI provider to process images.
func NewHandler(
	publisher *publisher.Publisher,
	uploader *object.Uploader,
	aiProvider ai.ImageAI,
) *Handler {
	return &Handler{
		converter: internalimg.NewConverter(),
		ai:        aiProvider,
		publisher: publisher,
		uploader:  uploader,
	}
}

// Message is the callback function that will be called when a message is received on the subscribed MQTT topic.
// client: The MQTT client that received the message.
// msg: The MQTT message that was received.
func (h *Handler) Message(_ mqtt.Client, msg mqtt.Message) {
	logrus.Debugf("received message on topic: %s, payload size: %d bytes", msg.Topic(), len(msg.Payload()))

	img, cErr := h.converter.FromPayload(msg.Payload())
	if cErr != nil {
		logrus.Errorf("failed to convert image: %v", cErr)
		return
	}
	bytes, eErr := h.converter.Encode(img)
	if eErr != nil {
		logrus.Errorf("failed to encode image: %v", eErr)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), processingTimeout)
	defer cancel()
	var wg sync.WaitGroup
	//nolint:mnd // 2 concurrent operations: upload and process
	wg.Add(2)

	go func() {
		defer wg.Done()
		h.uploader.Upload(ctx, bytes)
	}()

	go func() {
		defer wg.Done()
		h.processImage(ctx, bytes)
	}()

	wg.Wait()
}

// processImage takes an image, encodes it, processes it using the AI provider, and publishes the result.
// ctx: The context for the operation.
// data: The byte slice representing the encoded image to be processed.
func (h *Handler) processImage(ctx context.Context, data []byte) {
	result, piErr := h.ai.ProcessImage(ctx, data)
	if piErr != nil {
		logrus.Errorf("failed to process image: %v", piErr)
		return
	}
	logrus.Debugf("processed meter value: %s", *result)

	pErr := h.publisher.Publish(h.publisher.Topics.Status, false, *result)
	if pErr != nil {
		logrus.Errorf("failed to publish meter value: %v", pErr)
	} else {
		logrus.Infof("published meter value: %s", *result)
	}
}
