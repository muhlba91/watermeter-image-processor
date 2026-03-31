package publisher

import (
	"encoding/json"
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"

	"github.com/muhlba91/watermeter-image-processor/cmd/configuration"
	"github.com/muhlba91/watermeter-image-processor/internal/homeassistant"
)

// Publisher is responsible for publishing messages to MQTT topics and managing the MQTT client connection for publishing.
type Publisher struct {
	// Client is the MQTT client used for publishing to topics and receiving messages.
	client mqtt.Client
	// meter contains the information about the water meter, which can be used for device information in MQTT discovery messages for Home Assistant.
	meter *configuration.WaterMeter
	// Topics contains the MQTT topics for subscription and publication, which are generated based on the configuration.
	Topics *configuration.Topics
}

// NewPublisher initializes a new MQTT publisher by creating a new MQTT client with the provided configuration and connecting to the broker.
// configuration: The configuration for the MQTT publisher, including broker address, client ID, username, and password.
func NewPublisher(configuration *configuration.Data) (*Publisher, error) {
	topics := configuration.Topics()

	return &Publisher{
		meter:  configuration.WaterMeter(),
		Topics: topics,
	}, nil
}

// SetClient sets the MQTT client for the publisher, allowing it to publish messages to the MQTT broker.
func (p *Publisher) SetClient(client mqtt.Client) {
	p.client = client
}

// PublishInitialStatus publishes the initial status of the water meter processor to the MQTT broker, indicating that it is online and ready to receive messages.
func (p *Publisher) PublishInitialStatus() {
	discovery := homeassistant.Discovery{
		Name:                "Usage",
		UniqueID:            fmt.Sprintf("%s_usage", p.meter.Identifier),
		StateTopic:          p.Topics.Publish,
		AvailabilityTopic:   p.Topics.Status,
		PayloadAvailable:    "online",
		PayloadNotAvailable: "offline",
		UnitOfMeasurement:   "m³",
		DeviceClass:         "water",
		StateClass:          "total_increasing",
		Device: homeassistant.Device{
			Identifiers: []string{p.meter.Identifier},
			Name:        p.meter.Name,
			Model:       p.meter.Model,
		},
	}
	payload, err := json.Marshal(discovery)
	if err != nil {
		logrus.Errorf("failed to marshal discovery message: %v", err)
	}

	dErr := p.Publish(p.Topics.Discovery, true, string(payload))
	if dErr != nil {
		logrus.Errorf("failed to publish discovery: %v", dErr)
	} else {
		logrus.Infof("published discovery: %s", p.meter.Identifier)
	}

	sErr := p.Publish(p.Topics.Status, true, "online")
	if sErr != nil {
		logrus.Errorf("failed to publish availability: %v", sErr)
	} else {
		logrus.Infof("published availability: %s", p.meter.Identifier)
	}
}

// Publish is a helper function that publishes a message to a specified MQTT topic with the given payload and retain flag.
// topic: The MQTT topic to publish the message to.
// retain: A boolean flag indicating whether the message should be retained by the MQTT broker.
// payload: The payload of the message to be published, which can be of any type that can be marshaled to JSON.
func (p *Publisher) Publish(topic string, retain bool, payload string) error {
	logrus.Debugf("publishing to topic %s with payload: %v", topic, payload)
	token := p.client.Publish(topic, publishingQoS, retain, payload)
	token.Wait()

	return token.Error()
}
