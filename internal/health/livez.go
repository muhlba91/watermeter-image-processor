package health

import (
	"fmt"
	"net/http"
)

// livezHandler handles the /livez endpoint and returns the liveness status of the application, indicating whether it is running and responsive.
// w: The HTTP response writer.
func (s *Server) livezHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"up"}`)
}
