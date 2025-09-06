package middlewares

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func OTel(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		opName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		handler := otelhttp.NewHandler(next, opName)
		handler.ServeHTTP(w, r)
	})
}
