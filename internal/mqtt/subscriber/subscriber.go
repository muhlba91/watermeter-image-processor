package subscriber

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"

	"github.com/muhlba91/watermeter-image-processor/pkg/handler"
)

// Subscriber is responsible for subscribing to MQTT topics and handling incoming messages.
type Subscriber struct {
	// Client is the MQTT client used for subscribing to topics and receiving messages.
	client mqtt.Client
	// handler is the handler for processing incoming MQTT messages when a message is received on the subscribed topic.
	handler *handler.Handler
}

// NewSubscriber initializes a new MQTT subscriber by creating a new MQTT client with the provided configuration and connecting to the broker.
// configuration: The configuration for the MQTT subscriber, including broker address, client ID, username, and password.
// handler: The handler for processing incoming MQTT messages.
func NewSubscriber(handler *handler.Handler) (*Subscriber, error) {
	return &Subscriber{
		handler: handler,
	}, nil
}

// SetClient sets the MQTT client for the subscriber, allowing it to subscribe to topics and receive messages.
func (s *Subscriber) SetClient(client mqtt.Client) {
	s.client = client
}

// Subscribe subscribes the MQTT client to the specified topic with the provided handler for processing incoming messages.
// topic: The MQTT topic to subscribe to for receiving messages.
// qos: The Quality of Service level for the MQTT subscription, determining the guarantee of message delivery.
func (s *Subscriber) Subscribe(topic string, qos byte) {
	logrus.Debugf("subscribing to topic: %s (QoS: %d)", topic, qos)
	s.client.Subscribe(topic, qos, s.handler.Message)
}
