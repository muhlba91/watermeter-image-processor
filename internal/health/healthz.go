package health

import (
	"encoding/json"
	"net/http"
)

// healthzHandler handles the /healthz endpoint and returns the health status of the application, including the status of its components.
// w: The HTTP response writer.
func (s *Server) healthzHandler(w http.ResponseWriter, _ *http.Request) {
	status := http.StatusOK
	components := map[string]string{
		"mqtt":    "up",
		"ai":      "up",
		"storage": "up",
	}

	if !s.pubsub.IsConnected() {
		status = http.StatusServiceUnavailable
		components["mqtt"] = "down"
	}

	if !s.aiProvider.HealthCheck() {
		status = http.StatusServiceUnavailable
		components["ai"] = "down"
	}

	if !s.uploader.HealthCheck() {
		status = http.StatusServiceUnavailable
		components["storage"] = "down"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": components,
	})
}
