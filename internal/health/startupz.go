package health

import (
	"context"
	"encoding/json"
	"net/http"
)

// startupzHandler handles the /startupz endpoint and returns the startup status of the application, including the status of its components.
// w: The HTTP response writer.
func (s *Server) startupzHandler(w http.ResponseWriter, _ *http.Request) {
	status := http.StatusOK
	components := map[string]string{
		"mqtt":    valueUp,
		"ai":      valueUp,
		"storage": valueUp,
	}

	if !s.pubsub.IsConnected() {
		status = http.StatusServiceUnavailable
		components["mqtt"] = valueDown
	}

	if !s.aiProvider.HealthCheck() || !s.aiProvider.CheckModel(context.Background()) {
		status = http.StatusServiceUnavailable
		components["ai"] = valueDown
	}

	if !s.uploader.HealthCheck() {
		status = http.StatusServiceUnavailable
		components["storage"] = valueDown
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": components,
	})
}
