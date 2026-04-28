package schema

import (
	"log/slog"
	"net/http"
	luaApi "smart-pc-agent/internal/lib/lua-api"

	"github.com/MaxRomanov007/smart-pc-go-lib/api/response"
	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	"github.com/go-chi/render"
)

func New(log *slog.Logger, registry *luaApi.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "http-server.handlers.api.schema"
		log := log.With(sl.Op(op), sl.ReqID(r))

		schema := registry.Schema()
		log.Debug("got schema", slog.Any("schema", schema))
		render.JSON(w, r, response.OK(&schema))
	}
}
