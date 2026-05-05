package redirectAuth

import (
	"log/slog"
	"net/http"

	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
)

func New(log *slog.Logger, redirectURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "http-server.handlers.waker.redirect-auth"
		log := log.With(sl.Op(op), sl.ReqID(r))

		log.Info("redirecting auth request")

		http.Redirect(w, r, redirectURL+"?"+r.URL.Query().Encode(), http.StatusPermanentRedirect)
	}
}
