package authorization

import (
	"fmt"
	"io"
	"net/http"
)

// callbackHandler implements http.Handler for OAuth2 callback endpoint.
// It validates the state parameter and extracts the authorization code from the callback URL.
type callbackHandler struct {
	state          string        // Expected state parameter for CSRF protection
	codeChan       chan<- string // Channel to send the extracted authorization code
	errChan        chan<- error  // Channel to send any processing errors
	successMessage string        // Message displayed to user upon successful authorization
}

// newCallbackHandler creates a new callbackHandler instance.
// The handler validates the state parameter matches the expected value and
// extracts the authorization code from the callback URL query parameters.
func newCallbackHandler(
	state string,
	codeChan chan<- string,
	errChan chan<- error,
) *callbackHandler {
	return &callbackHandler{
		state:          state,
		codeChan:       codeChan,
		errChan:        errChan,
		successMessage: "Авторизация успешна! Закройте вкладку и вернитесь в CLI.",
	}
}

// ServeHTTP handles the OAuth2 callback request.
// Validates the state parameter, extracts the authorization code, and sends it through the channel.
// Returns HTTP 400 for invalid state or missing code parameters.
func (h *callbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const op = "lib.authorization.callback-handler.ServeHTTP"

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	state := query.Get("state")
	code := query.Get("code")

	if state != h.state {
		http.Error(w, "Invalid state", http.StatusBadRequest)
		h.errChan <- fmt.Errorf("%s: invalid state", op)
		return
	}

	if code == "" {
		http.Error(w, "No code", http.StatusBadRequest)
		h.errChan <- fmt.Errorf("%s: no code", op)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.WriteString(w, h.successMessage)

	select {
	case h.codeChan <- code:
	default:
		h.errChan <- fmt.Errorf("%s: failed to send code - channel closed", op)
	}
}
