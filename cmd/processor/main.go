package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/muhlba91/watermeter-image-processor/cmd/configuration"
	"github.com/muhlba91/watermeter-image-processor/cmd/logging"
	"github.com/muhlba91/watermeter-image-processor/internal/health"
	"github.com/muhlba91/watermeter-image-processor/internal/image/ai/gemini"
	"github.com/muhlba91/watermeter-image-processor/internal/mqtt"
	"github.com/muhlba91/watermeter-image-processor/internal/mqtt/publisher"
	"github.com/muhlba91/watermeter-image-processor/internal/mqtt/subscriber"
	"github.com/muhlba91/watermeter-image-processor/internal/scaleway/object"
	"github.com/muhlba91/watermeter-image-processor/pkg/handler"
)

// Version and Gitsha of the current build.
//
//nolint:gochecknoglobals // These variables are set at build time using ldflags.
var (
	Version = "local"
	Gitsha  = "?"
)

// main is the entry point of the application.
func main() {
	logging.Init()

	logrus.Infof("version: %s (%s)", Version, Gitsha)

	cfg := configuration.Init()

	uploader, uErr := object.NewUploader(&cfg)
	if uErr != nil {
		logrus.Fatalf("failed to initialize scaleway uploader: %v", uErr)
	}
	aiProvider, err := gemini.NewGemini(&cfg)
	if err != nil {
		logrus.Fatalf("failed to initialize image processor: %v", err)
	}

	publisher, pErr := publisher.NewPublisher(&cfg)
	if pErr != nil {
		logrus.Fatalf("failed to initialize mqtt publisher: %v", pErr)
	}
	handler := handler.NewHandler(publisher, uploader, aiProvider)
	subscriber, err := subscriber.NewSubscriber(handler)
	if err != nil {
		logrus.Fatalf("failed to initialize mqtt subscriber: %v", err)
	}

	pubsub, err := mqtt.NewPubSub(&cfg, publisher, subscriber)
	if err != nil {
		logrus.Fatalf("failed to initialize pubsub providers: %v", err)
	}
	publisher.PublishInitialStatus()

	healthServer := health.NewServer(&cfg, uploader, aiProvider, pubsub)
	healthServer.Start()

	aiProvider.CheckModel(context.Background())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	signal.Notify(sig, syscall.SIGTERM)

	<-sig
	logrus.Info("shutting down...")
	pubsub.Disconnect()
	healthServer.Stop()
}
