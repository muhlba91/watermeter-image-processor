package mqtt

// disconnectTimeout is the time in milliseconds to wait before forcefully disconnecting the MQTT client.
const disconnectTimeout = 1000

// keepalive is the time in seconds to send a ping to the MQTT broker to keep the connection alive.
const keepalive = 60

// subscriptionQoS is the Quality of Service level to use when subscribing to the MQTT topic.
const subscriptionQoS byte = 0
