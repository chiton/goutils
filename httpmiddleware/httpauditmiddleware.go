package httpmiddleware

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"gitlab.edgecastcdn.net/edgecast/web-platform/identity/brokerclient"
	"gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/auditeventspublisher"
	"gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/integrationevents"
	"gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/jwtverifier"
	"gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/log"
)

const (
	ClaimClientId       = "client_id"
	ClaimClientTenantId = "client_tenant_id"
	ClaimSubjectId      = "sub"

	HeaderXForwardedProto = "X-Forwarded-Proto"
	HeaderXForwardedFor   = "X-Forwarded-For"
	HeaderXForwardedHost  = "X-Forwarded-Host"
	HeaderAccept          = "Accept"
	HeaderAcceptEncoding  = "Accept-Encoding"
	HeaderUserAgent       = "User-Agent"
	HeaderReferer         = "Referer"
	HeaderXLocation       = "X-Location"
)

var relevantHeaders = []string{
	HeaderXForwardedProto,
	HeaderXForwardedFor,
	HeaderXForwardedHost,
	HeaderAccept,
	HeaderAcceptEncoding,
	HeaderUserAgent,
	HeaderReferer,
	HeaderXLocation,
}

// HttpAuditMiddleware reads header values and enriches a publisher with those attributes. The publisher is
// put into a new context where it can be retrieved later on to publish audit events.
func HttpAuditMiddleware(publisher brokerclient.Publisher, source string) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			attrs := []integrationevents.Attribute{}

			// Fill claims data
			reqCtx := r.Context()
			claims := jwtverifier.ClaimsFromContext(reqCtx)
			if claims == nil {
				log := log.FromContext(reqCtx)
				log.Warnf("Could not get claims from context")
			} else {
				if clientId, found := claims[ClaimClientId].(string); found {
					id, err := uuid.Parse(clientId)
					if err == nil {
						attrs = append(attrs, integrationevents.Attribute{
							Key:   integrationevents.AttrKeyClientId,
							Value: id,
						})
					}
				}

				if clientTenantId, found := claims[ClaimClientTenantId].(string); found {
					id, err := uuid.Parse(clientTenantId)
					if err == nil {
						attrs = append(attrs, integrationevents.Attribute{
							Key:   integrationevents.AttrKeyClientTenantId,
							Value: id,
						})
					}
				}

				if subjectId, found := claims[ClaimSubjectId].(string); found {
					id, err := uuid.Parse(subjectId)
					if err == nil {
						attrs = append(attrs, integrationevents.Attribute{
							Key:   integrationevents.AttrKeyUsertId,
							Value: id,
						})
					}
				}
			}

			// Fill headers data
			for _, relevantHeader := range relevantHeaders {
				if headerValue, ok := r.Header[relevantHeader]; ok {
					attrs = appendAttribute(attrs, relevantHeader, firstOrEmpty(headerValue))
				}
			}

			eventsPublisher := auditeventspublisher.NewAuditEventsPublisher(publisher, attrs, source)
			newCtx := auditeventspublisher.NewContext(r.Context(), eventsPublisher)
			newRequest := r.WithContext(newCtx)

			next.ServeHTTP(w, newRequest)
		}
		return http.HandlerFunc(fn)
	}
}

func appendAttribute(attrs []integrationevents.Attribute, key, value string) []integrationevents.Attribute {
	switch key {
	case HeaderUserAgent:
		attrs = append(attrs, integrationevents.Attribute{
			Key:   integrationevents.AttrKeyClientRawUserAgent,
			Value: value,
		})
	case HeaderXForwardedFor:
		attrs = append(attrs, integrationevents.Attribute{
			Key:   integrationevents.AttrKeyClientIp,
			Value: value,
		})
	case HeaderXForwardedProto:
		attrs = append(attrs, integrationevents.Attribute{
			Key:   integrationevents.AttrKeyClientProto,
			Value: value,
		})
	case HeaderXForwardedHost:
		attrs = append(attrs, integrationevents.Attribute{
			Key:   integrationevents.AttrKeyClientHost,
			Value: value,
		})
	case HeaderReferer:
		attrs = append(attrs, integrationevents.Attribute{
			Key:   integrationevents.AttrKeyClientReferer,
			Value: value,
		})
	case HeaderAccept:
		attrs = append(attrs, integrationevents.Attribute{
			Key:   integrationevents.AttrKeyClientAccept,
			Value: value,
		})
	case HeaderAcceptEncoding:
		attrs = append(attrs, integrationevents.Attribute{
			Key:   integrationevents.AttrKeyClientAcceptEncoding,
			Value: value,
		})
	case HeaderXLocation:
		attrs = appendLocationAttributes(attrs, value)
	}

	return attrs
}

func appendLocationAttributes(attrs []integrationevents.Attribute, location string) []integrationevents.Attribute {
	values := strings.Split(location, ",")
	for _, value := range values {
		keyValue := strings.Split(value, "=")
		key := strings.ToLower(strings.TrimSpace(keyValue[0]))
		switch key {
		case "continent":
			attrs = append(attrs, integrationevents.Attribute{
				Key:   integrationevents.AttrKeyClientContinent,
				Value: strings.TrimSpace(keyValue[1]),
			})
		case "country":
			attrs = append(attrs, integrationevents.Attribute{
				Key:   integrationevents.AttrKeyClientCountry,
				Value: strings.TrimSpace(keyValue[1]),
			})
		case "city":
			attrs = append(attrs, integrationevents.Attribute{
				Key:   integrationevents.AttrKeyClientCity,
				Value: strings.TrimSpace(keyValue[1]),
			})
		case "lat":
			attrs = append(attrs, integrationevents.Attribute{
				Key:   integrationevents.AttrKeyClientLat,
				Value: strings.TrimSpace(keyValue[1]),
			})
		case "long":
			attrs = append(attrs, integrationevents.Attribute{
				Key:   integrationevents.AttrKeyClientLong,
				Value: strings.TrimSpace(keyValue[1]),
			})
		}
	}

	return attrs
}
