package request

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/MaxRomanov007/smart-pc-go-lib/api/response"
	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type ctxKey string

const requestKey ctxKey = "request"

func New[T any](
	log *slog.Logger,
	validate *validator.Validate,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			const op = "middlewares.request"
			log := log.With(sl.Op(op), sl.ReqID(r))

			var req T
			if err := render.DecodeJSON(r.Body, &req); err != nil {
				log.Error("failed to decode request body", sl.Err(err))
				render.JSON(w, r, response.InternalError())
				return
			}

			log.Debug("request decoded", slog.Any("request", req))

			if err := validate.Struct(req); err != nil {
				if err, ok := errors.AsType[validator.ValidationErrors](err); ok {
					log.Warn("invalid request body", sl.Err(err))
					render.JSON(w, r, response.ValidationError(err))
					return
				}

				log.Error("failed to validate request", sl.Err(err))
				render.JSON(w, r, response.InternalError())
				return
			}

			log.Debug("request validated successfully")

			ctx := context.WithValue(r.Context(), requestKey, req)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func FromContext[T any](ctx context.Context) (T, bool) {
	val := ctx.Value(requestKey)
	req, ok := val.(T)
	return req, ok
}

func MustFromContext[T any](ctx context.Context) T {
	const op = "middlewares.request.MustFromContext"

	req, ok := FromContext[T](ctx)
	if !ok {
		panic(
			fmt.Errorf(
				"%s: can not get request from context, looks like you forgot to use middleware",
				op,
			),
		)
	}
	return req
}

func MustGet[T any](r *http.Request) T {
	return MustFromContext[T](r.Context())
}
