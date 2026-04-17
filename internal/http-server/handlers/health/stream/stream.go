package stream

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
)

func New(log *slog.Logger, shutdownCtx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.health.stream"

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		go func() {
			select {
			case <-shutdownCtx.Done():
				cancel()
			case <-ctx.Done():
			}
		}()

		log := log.With(sl.Op(op), sl.ReqID(r))

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		rc := http.NewResponseController(w)

		if err := rc.SetWriteDeadline(time.Time{}); err != nil {
			log.Warn("failed to set write deadline", sl.Err(err))
		}

		log.Info("start sending health event")

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		if err := sendPingEvent(w, rc); err != nil {
			log.Warn("failed to send ping event", sl.Err(err))
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := sendPingEvent(w, rc); err != nil {
					log.Warn("failed to send ping event", sl.Err(err))
					return
				}
			}
		}
	}
}
