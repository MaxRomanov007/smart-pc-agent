package redirect

import (
	"log/slog"
	"net/http"

	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
)

func New(log *slog.Logger, redirectURL string, code int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "http-server.handlers.redirect"
		log := log.With(sl.Op(op), sl.ReqID(r))

		log.Info("redirecting request")

		http.Redirect(w, r, redirectURL+"?"+r.URL.Query().Encode(), code)
	}
}
