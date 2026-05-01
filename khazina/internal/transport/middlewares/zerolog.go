package middlewares

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/rs/zerolog/log"
)

func ZeroLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		reqID := middleware.GetReqID(r.Context())

		logger := log.With().
			Str("req_id", reqID).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("remote", r.RemoteAddr).
			Str("user_agent", r.UserAgent()).
			Logger()

		ctx := logger.WithContext(r.Context())

		next.ServeHTTP(ww, r.WithContext(ctx))

		status := ww.Status()
		event := logger.Info()

		if status >= 500 {
			event = logger.Error()
		} else if status >= 400 {
			event = logger.Warn()
		}

		event.
			Int("status", status).
			Int("bytes", ww.BytesWritten()).
			Dur("duration", time.Since(start)).
			Msg("http_request")
	})
}
