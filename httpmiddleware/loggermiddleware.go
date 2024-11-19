package httpmiddleware

import (
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/correlation"
	internallog "gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/log"
	"go.uber.org/zap"
)

const AmznApigatewayIdHeader = "X-Amzn-Apigateway-Api-Id"
const EcPopHeader = "X-Ec-Pop"
const EcSessionIdHeader = "X-Ec-Session-Id"
const EcUUIDHeader = "X-Ec-Uuid"
const HostHeader = "X-Host"

var traceIdRegex *regexp.Regexp

// LoggerMiddleware is a middleware that logs the start and end of each request, along
// with some useful data about what was requested, what the response status was,
// and how long it took to return.
func LoggerMiddleware(l *zap.SugaredLogger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			correlationId := correlation.FromContext(r.Context())
			clientCorrelationId := correlation.FromContextWithClientCorrelationId(r.Context())

			newLogger := l.With("correlation-id", correlationId)
			newRequest := r.WithContext(internallog.NewContext(r.Context(), newLogger))

			withs := []any{
				"path", r.URL.Path,
				AmznApigatewayIdHeader, firstOrEmpty(r.Header[AmznApigatewayIdHeader]),
				EcPopHeader, firstOrEmpty(r.Header[EcPopHeader]),
				EcSessionIdHeader, firstOrEmpty(r.Header[EcSessionIdHeader]),
				EcUUIDHeader, firstOrEmpty(r.Header[EcUUIDHeader]),
				HostHeader, firstOrEmpty(r.Header[HostHeader]),
			}

			if clientCorrelationId != "" {
				withs = append(withs, "client-correlation-id", clientCorrelationId)
			}

			newLogger.Infow(fmt.Sprintf("Begin %s %s", r.Method, r.URL.Path), withs...)

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			t1 := time.Now()

			defer func() {
				newLogger.Infow(fmt.Sprintf("End %s %s with status %d", r.Method, r.URL.Path, ww.Status()),
					"status", ww.Status(),
					"duration", time.Since(t1))
			}()

			next.ServeHTTP(ww, newRequest)
		}
		return http.HandlerFunc(fn)
	}
}

func firstOrEmpty(s []string) string {
	if len(s) >= 1 {
		return s[0]
	}

	return ""
}
