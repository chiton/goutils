package httpmiddleware

import (
	"net/http"
	"runtime/debug"

	"github.com/teris-io/shortid"
	logger "gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/log"
)

// PanicRecoveryMiddleware recovers from unexpected panics and logs it as an error.
// Use this as one of the outermost middleware to cast the widest net for recovering from panics
// but must be placed after CorrelationId and Logger middlewares.
// The input parameter, panicResponder, is a function that should write an error response to the http.ResponseWriter
// and it is invoked if and only if a panic is recovered. An errorId is generated and passed to the input func
// so that it can be returned to the client.
func PanicRecoveryMiddleware(panicResponder func(w http.ResponseWriter, r *http.Request, errorId string)) func(next http.Handler) http.Handler {
	middle := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				err := recover()

				if err != nil {
					log := logger.FromContext(r.Context())
					errorId, _ := shortid.Generate()
					log.Errorw("unexpected panic occurred",
						"error", err,
						"errorId", errorId,
						"stacktrace", string(debug.Stack()))

					if panicResponder != nil {
						panicResponder(w, r, errorId)
					}
				}
			}()

			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
	return middle
}
