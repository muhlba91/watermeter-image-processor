package mqtt

import (
	"crypto/tls"
	"net/url"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"

	"github.com/muhlba91/watermeter-image-processor/cmd/configuration"
	"github.com/muhlba91/watermeter-image-processor/internal/mqtt/publisher"
	"github.com/muhlba91/watermeter-image-processor/internal/mqtt/subscriber"
)

// PubSub is responsible for managing both the MQTT subscriber and publisher clients, allowing for subscribing to topics and publishing messages to the MQTT broker.
type PubSub struct {
	// Subscriber is the MQTT client used for subscribing to topics.
	Subscriber *subscriber.Subscriber
	// Publisher is the MQTT client used for publishing to topics.
	Publisher *publisher.Publisher
	// client is the underlying MQTT client that both the subscriber and publisher use to connect to the broker.
	client mqtt.Client
}

// NewPubSub creates and initializes a new PubSub instance, setting up the MQTT subscriber and publisher clients based on the provided configuration and handler.
// configuration: The configuration for the MQTT publisher, including broker address, client ID, username, and password.
// publisher: The MQTT publisher to use for publishing messages to the broker.
// subscriber: The MQTT subscriber to use for subscribing to topics and handling incoming messages.
func NewPubSub(
	configuration *configuration.Data,
	publisher *publisher.Publisher,
	subscriber *subscriber.Subscriber,
) (*PubSub, error) {
	options := getOptions(configuration)
	addHandlers(options, configuration, subscriber)
	client := mqtt.NewClient(options)

	subscriber.SetClient(client)
	publisher.SetClient(client)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	return &PubSub{
		Subscriber: subscriber,
		Publisher:  publisher,
		client:     client,
	}, nil
}

// IsConnected checks if the MQTT client is currently connected to the broker and returns a boolean value indicating the connection status.
func (s *PubSub) IsConnected() bool {
	return s.client.IsConnected()
}

// Disconnect disconnects both the MQTT subscriber and publisher clients from the broker, waiting for a specified timeout before forcefully disconnecting.
func (s *PubSub) Disconnect() {
	logrus.Debug("disconnecting from broker")
	s.client.Disconnect(disconnectTimeout)
}

// getOptions creates and returns a new MQTT client options struct based on the provided configuration.
// configuration: The configuration containing the parameters for the MQTT client options, such as broker address and authentication details.
func getOptions(configuration *configuration.Data) *mqtt.ClientOptions {
	options := mqtt.NewClientOptions()

	options.AddBroker(configuration.BrokerAddress)

	setAuthentication(options, configuration)

	options.SetBinaryWill(configuration.Topics().Status, []byte("offline"), 1, true)
	options.SetCleanSession(true)
	options.SetOrderMatters(false)

	options.SetConnectTimeout(time.Second)
	options.SetWriteTimeout(time.Second)
	options.SetPingTimeout(time.Second)
	options.SetKeepAlive(keepalive)
	options.SetConnectRetry(true)
	options.SetAutoReconnect(true)

	return options
}

// addHandlers adds the necessary handlers to the MQTT client options based on the provided configuration.
// options: The MQTT client options to add handlers to.
// configuration: The configuration containing the parameters for the MQTT client options, such as broker address and authentication details.
// subscriber: The MQTT subscriber to use for handling incoming messages and connection events.
func addHandlers(options *mqtt.ClientOptions, configuration *configuration.Data, subscriber *subscriber.Subscriber) {
	options.SetOnConnectHandler(func(_ mqtt.Client) {
		logrus.Infof("connected to broker: %s", configuration.BrokerAddress)
		subscriber.Subscribe(configuration.Topics().Subscription, subscriptionQoS)
	})

	options.SetConnectionAttemptHandler(func(broker *url.URL, tlsCfg *tls.Config) *tls.Config {
		logrus.Debugf("connecting to broker: %s", broker.String())
		return tlsCfg
	})

	options.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		logrus.Errorf("connection to broker lost: %s, error: %v", configuration.BrokerAddress, err)
	})
}

// setAuthentication sets the client ID, username, and password for the MQTT client options based on the provided configuration.
// options: The MQTT client options to set the authentication parameters on.
// configuration: The configuration containing the authentication parameters to set.
func setAuthentication(options *mqtt.ClientOptions, configuration *configuration.Data) {
	if configuration.BrokerClientID != nil {
		options.SetClientID(*configuration.BrokerClientID)
	}
	if configuration.BrokerUsername != nil {
		options.SetUsername(*configuration.BrokerUsername)
	}
	if configuration.BrokerPassword != nil {
		options.SetPassword(*configuration.BrokerPassword)
	}
}
