package homeassistant

// Discovery represents the structure of the MQTT discovery message that is sent to Home Assistant to automatically discover and configure the water meter sensor.
type Discovery struct {
	Name                string `json:"name"`
	UniqueID            string `json:"unique_id"`
	StateTopic          string `json:"state_topic"`
	AvailabilityTopic   string `json:"availability_topic"`
	PayloadAvailable    string `json:"payload_available"`
	PayloadNotAvailable string `json:"payload_not_available"`
	UnitOfMeasurement   string `json:"unit_of_measurement"`
	DeviceClass         string `json:"device_class"`
	StateClass          string `json:"state_class"`
	Device              Device `json:"device"`
}

// Device represents the structure of the device information that is included in the MQTT discovery message for Home Assistant, providing details about the water meter device such as its identifiers, name, and model.
type Device struct {
	Identifiers []string `json:"identifiers"`
	Name        string   `json:"name"`
	Model       string   `json:"model"`
}
