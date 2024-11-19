package httpmiddleware

import (
	"net/http"

	"github.com/teris-io/shortid"
	"gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/correlation"
)

const (
	CorrelationIdHeader       = "X-Correlation-Id"
	ClientCorrelationIdHeader = "X-Client-Correlation-Id"
)

/*
CorrelationIdMiddleware generates a correlation using a new shortid for each request.
Adds the X-Correlation-Id header to the response.

It also reads an X-Client-Correlation-Id from the request headers and saves that in the context
as a client correlation id value.
*/
func CorrelationIdMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		correlationId := firstOrEmpty(r.Header[CorrelationIdHeader])
		if correlationId == "" {
			correlationId, _ = shortid.Generate()
		}

		newContext := correlation.NewContext(r.Context(), correlationId)

		// the X-Correlation-Id header in the response is the server-generated ID
		// or the one generated and passed to us by the API Gateway. It is not to be confused with the header
		// of the same name in the request.
		w.Header().Add(CorrelationIdHeader, correlationId)

		// read the client correlation id
		clientCorrelationId := firstOrEmpty(r.Header[ClientCorrelationIdHeader])
		if clientCorrelationId != "" {
			newContext = correlation.NewContextWithClientCorrelationId(newContext, clientCorrelationId)
			w.Header().Add(ClientCorrelationIdHeader, clientCorrelationId)
		}

		newRequest := r.WithContext(newContext)

		next.ServeHTTP(w, newRequest)
	}
	return http.HandlerFunc(fn)
}
