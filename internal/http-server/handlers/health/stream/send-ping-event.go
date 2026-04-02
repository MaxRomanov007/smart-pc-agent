package stream

import (
	"fmt"
	"net/http"
)

const pingEvent = "event: ping\ndata: {}\n\n"

func sendPingEvent(w http.ResponseWriter, rc *http.ResponseController) error {
	const op = "handlers.health.stream.sendPingEvent"

	_, err := fmt.Fprintf(w, pingEvent)
	if err != nil {
		return fmt.Errorf("%s: failed to print event: %w", op, err)
	}
	if err := rc.Flush(); err != nil {
		return fmt.Errorf("%s: failed to flush: %w", op, err)
	}

	return nil
}
