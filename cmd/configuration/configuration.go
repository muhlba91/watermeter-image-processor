package configuration

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	log "github.com/sirupsen/logrus"
)

// Data defines the configuration for the processor init command, which is read from environment variables.
type Data struct {
	// MeterID is the ID of the water meter to be processed
	MeterID string `env:"METER_ID" envDefault:"water-meter"`
	// MeterName is the name of the water meter to be processed
	MeterName string `env:"METER_NAME" envDefault:"Water Meter"`
	// MeterModel is the model of the water meter to be processed
	MeterModel string `env:"METER_MODEL" envDefault:"ESP32 Water Meter"`
	// BrokerAddress is the address of the MQTT broker to connect to
	BrokerAddress string `env:"BROKER_ADDRESS" envDefault:"tcp://localhost:1883"`
	// BrokerTopicSubscriptionTemplate is the template for the MQTT topic to subscribe to for receiving water meter images
	BrokerTopicSubscriptionTemplate string `env:"BROKER_TOPIC_SUBSCRIPTION_TEMPLATE" envDefault:"tele/%s/image"`
	// BrokerTopicPublishTemplate is the template for the MQTT topic to publish the processed water meter usage data to
	BrokerTopicPublishTemplate string `env:"BROKER_TOPIC_PUBLISH_TEMPLATE" envDefault:"stat/%s/water/usage/state"`
	// BrokerClientID is the client ID to use when connecting to the MQTT broker
	BrokerClientID *string `env:"BROKER_CLIENT_ID"`
	// BrokerUsername is the username to use when connecting to the MQTT broker
	BrokerUsername *string `env:"BROKER_USERNAME"`
	// BrokerPassword is the password to use when connecting to the MQTT broker
	BrokerPassword *string `env:"BROKER_PASSWORD"`
	// GeminiAPIKey is the API key for the Gemini server to connect to for image processing
	GeminiAPIKey string `env:"GEMINI_API_KEY"`
	// GeminiModel is the name of the Gemini model to be used for image processing
	GeminiModel string `env:"GEMINI_MODEL" envDefault:"gemini-3.5-flash-lite"`
	// SCWRegion is the Scaleway region to use for the S3 client, which is required for connecting to Scaleway's S3-compatible object storage service
	SCWRegion *string `env:"SCW_REGION" envDefault:"fr-par"`
	// SCWAccessKey is the access key to use when connecting to Scaleway's S3-compatible object storage service, which is required for authentication
	SCWAccessKey *string `env:"SCW_ACCESS_KEY"`
	// SCWSecretKey is the secret key to use when connecting to Scaleway's S3-compatible object storage service, which is required for authentication
	SCWSecretKey *string `env:"SCW_SECRET_KEY"`
	// SCWBucket is the name of the bucket to use when connecting to Scaleway's S3-compatible object storage service, which is required for storing the water meter images and processed data
	SCWBucket *string `env:"SCW_BUCKET"`
	// SCWBucketPath is the path within the bucket to use when connecting to Scaleway's S3-compatible object storage service, which is optional and defaults to "watermeter/"
	SCWBucketPath string `env:"SCW_BUCKET_PATH" envDefault:"watermeter/%s/"`
	// HealthzHost is the host for the healthz server to listen on
	HealthzHost string `env:"HEALTHZ_HOST" envDefault:"0.0.0.0"`
	// HealthzPort is the port for the healthz server to listen on
	HealthzPort int `env:"HEALTHZ_PORT" envDefault:"8080"`
}

// Topics defines the MQTT topics for subscription and publication, which are generated based on the configuration.
type Topics struct {
	// Subscription is the MQTT topic to subscribe to for receiving water meter images
	Subscription string
	// PublishReading is the MQTT topic to publish the processed water meter usage data to
	Publish string
	// Status is the MQTT topic to publish the status of the water meter processor to
	Status string
	// Discovery is the MQTT topic to publish the MQTT discovery message for Home Assistant
	Discovery string
}

// WaterMeter represents the structure of a water meter, including its unique identifier, name, and model, which can be used for device information in MQTT discovery messages for Home Assistant.
type WaterMeter struct {
	// Identifier is a unique identifier for the meter.
	Identifier string
	// Name is the name of the meter.
	Name string
	// Model is the model of the meter.
	Model string
}

// Init reads the configuration from environment variables and returns a Config struct.
func Init() Data {
	cfg := Data{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("error reading configuration from environment: %v", err)
	}
	return cfg
}

// Topics generates and returns the MQTT topics for subscription and publication based on the configuration.
func (d *Data) Topics() *Topics {
	return &Topics{
		Subscription: d.subscriptionTopic(),
		Publish:      d.publishReadingTopic(),
		Status:       d.statusTopic(),
		Discovery:    d.discoveryTopic(),
	}
}

// WaterMeter generates and returns a WaterMeter struct based on the configuration, which can be used for device information in MQTT discovery messages for Home Assistant.
func (d *Data) WaterMeter() *WaterMeter {
	return &WaterMeter{
		Identifier: d.MeterID,
		Name:       d.MeterName,
		Model:      d.MeterModel,
	}
}

// subscriptionTopic returns the MQTT topic to subscribe to for receiving water meter images, based on the BrokerTopicSubscriptionTemplate and MeterID.
func (d *Data) subscriptionTopic() string {
	return fmt.Sprintf(d.BrokerTopicSubscriptionTemplate, d.MeterID)
}

// publishReadingTopic returns the MQTT topic to publish the processed water meter usage data to, based on the BrokerTopicPublishTemplate and MeterID.
func (d *Data) publishReadingTopic() string {
	return fmt.Sprintf(d.BrokerTopicPublishTemplate, d.MeterID)
}

// statusTopic returns the MQTT topic to publish the status of the water meter processor to, based on the BrokerTopicPublishTemplate and MeterID.
func (d *Data) statusTopic() string {
	return fmt.Sprintf(d.BrokerTopicPublishTemplate, d.MeterID) + "_status"
}

// discoveryTopic returns the MQTT topic to publish the MQTT discovery message for Home Assistant, based on the BrokerTopicPublishTemplate and MeterID.
func (d *Data) discoveryTopic() string {
	return fmt.Sprintf("homeassistant/sensor/%s/usage/config", d.MeterID)
}
