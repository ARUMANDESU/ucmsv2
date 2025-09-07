package middlewares

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}

		t1 := time.Now()
		defer func() {
			logstr := fmt.Sprintf("HTTP Request Completed %s %s://%s%s %s from %s - %d %dB in %s",
				r.Method,
				scheme,
				r.Host,
				r.RequestURI,
				r.Proto,
				r.RemoteAddr,
				ww.Status(),
				ww.BytesWritten(),
				time.Since(t1),
			)
			logger := slog.With(
				slog.String("method", r.Method),
				slog.String("url", fmt.Sprintf("%s://%s%s", scheme, r.Host, r.RequestURI)),
				slog.String("proto", r.Proto),
				slog.String("remote_addr", r.RemoteAddr),
				slog.Int("status", ww.Status()),
				slog.Int("bytes", ww.BytesWritten()),
				slog.Duration("duration", time.Since(t1)),
			)

			if ww.Status() >= 500 {
				logger.ErrorContext(r.Context(), logstr)
			} else if ww.Status() >= 400 {
				logger.WarnContext(r.Context(), logstr)
			} else {
				logger.InfoContext(r.Context(), logstr)
			}
		}()

		next.ServeHTTP(ww, r)
	})
}
