package middlewares

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/rs/zerolog/log"
)

type responseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

func ZeroLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{ResponseWriter: w}
		reqID := middleware.GetReqID(r.Context())

		logger := log.With().
			Str("req_id", reqID).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("remote", r.RemoteAddr).
			Str("user_agent", r.UserAgent()).
			Logger()

		ctx := logger.WithContext(r.Context())
		next.ServeHTTP(rw, r.WithContext(ctx))

		event := logger.Info()
		if rw.status >= 500 {
			event = logger.Error()
		} else if rw.status >= 400 {
			event = logger.Warn()
		}
		event.
			Int("status", rw.status).
			Int("bytes", rw.bytes).
			Dur("duration", time.Since(start)).
			Msg("http_request")
	})
}
