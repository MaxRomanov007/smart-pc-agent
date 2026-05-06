package wakerCallback

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
)

type AuthChecker interface {
	IsAuthorized(ctx context.Context) (bool, error)
}

// New создаёт redirect-хендлер. onSuccess вызывается после успешного редиректа;
// передайте nil если уведомление не нужно.
func New(
	appCtx context.Context,
	log *slog.Logger,
	redirectURL string,
	code int,
	onSuccess func(),
	checker AuthChecker,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "http-server.handlers.redirect"
		log := log.With(sl.Op(op), sl.ReqID(r))

		log.Info("redirecting request")

		http.Redirect(w, r, redirectURL+"?"+r.URL.Query().Encode(), code)

		if onSuccess == nil {
			return
		}
		go func() {
			for latency := time.Second; latency < time.Minute; latency *= 2 {
				time.Sleep(latency)
				log.Info("checking authorization", slog.String("latency", latency.String()))

				isAuthorized, err := checker.IsAuthorized(appCtx)
				if err != nil {
					log.Error("failed to check auth status", sl.Err(err))
					continue
				}
				if isAuthorized {
					onSuccess()
					break
				}
			}
		}()
	}
}
