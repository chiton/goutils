package httpmiddleware

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.edgecastcdn.net/edgecast/web-platform/identity/brokerclient"
	"gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/auditeventspublisher"
	"gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/integrationevents"
	"gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/jwtverifier"
	"gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/testcat"
)

type mockResponseWriter struct {
	mock.Mock
}

func (rw *mockResponseWriter) Header() http.Header {
	return nil
}

func (rw *mockResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (rw *mockResponseWriter) WriteHeader(statusCode int) {
}

type testAuditEvent struct {
}

type mockPublisher struct {
	mock.Mock
	auditEvent *integrationevents.AuditLogEvent
}

var _ brokerclient.Publisher = &mockPublisher{}

func buildRequest(token, method, url string, headers map[string]string) (*http.Request, error) {
	bearer := "Bearer " + token
	// Create a new request using http
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	// add authorization header to the req
	req.Header.Add("Authorization", bearer)

	// add extra headers
	for key, val := range headers {
		req.Header.Add(key, val)
	}

	return req, nil
}

func (p *mockPublisher) Publish(ctx context.Context, msg *brokerclient.BrokerMessage) error {
	arguments := p.Called(ctx, msg)
	err, _ := arguments.Get(0).(error)

	var auditEvent integrationevents.AuditLogEvent
	if e := json.Unmarshal(msg.Content, &auditEvent); e != nil {
		panic(e)
	}
	p.auditEvent = &auditEvent

	return err
}

func TestHttpAuditMiddleware(t *testing.T) {
	testcat.CheckTestCategory(t, testcat.UnitTest)

	t.Run("should add events publisher into the context", func(t *testing.T) {
		publisher := &mockPublisher{}
		publisher.On("Publish", mock.Anything, mock.Anything).Return(nil)

		auditMid := HttpAuditMiddleware(publisher, "httpAuditMiddlewareTest")

		token := "eyJhbGciOiJSUzI1NiIsImtpZCI6IjI3MjlFRjY4MTYxQjFGQUQ1MkIzMTU2MjM4QkY2MUYxNzMwQjY5NzEiLCJ0eXAiOiJKV1QiLCJ4NXQiOiJKeW52YUJZYkg2MVNzeFZpT0w5aDhYTUxhWEUifQ.eyJuYmYiOjE2NTI3MzM1MDIsImV4cCI6MTY1MjczNzEwMiwiaXNzIjoiaHR0cHM6Ly9pZC1kZXYudmRtcy5pbyIsImF1ZCI6WyJpZC5hcGkiLCJodHRwczovL2lkLWRldi52ZG1zLmlvL3Jlc291cmNlcyJdLCJjbGllbnRfaWQiOiIwOWMxYzkxYS04NGJiLTRkYjQtODUyZC1iNDk2NzBjMmEzZDMiLCJqdGkiOiI4REI2OTRCQTEwOUZDOTRCOTQ2QTkwQkYzQjAyNTZERCIsImlhdCI6MTY1MjczMzUwMiwic2NvcGUiOlsiaWQuYXBpbS5hcGlzOnJlYWQiXX0.ZRxhhk0357HsihHQrqQBA4nIX4JlLJp3JwA4Pw9_ZJbBcORJmVZoTFfnSkcxBym2A1S_bGEBzS-Jo-MoY7YWkypy0YGVOk6bLehuPhk4r2NmYbIrxfLonIYWapJdKgLSNrhvOxoi6SJNObx_n2LOQgPmMbY_9CTnlL_jWsi12p0h2CITXmH7vDcDKo1Iy2goQrS0cLj_B9GJvQ1FiDMsl5G9tORA1D3VSoEUJ5hHRY4zl1Rjeu_jL7enwc5y1mPxk0wD5EflZQs7lG7z3-OHeIDLTr5VzHR5hTRXqBfiByGUi5VumZPHBDIMan7R5N2zBPel2dzYhhL94olCtrARTg"
		url := "http://0.0.0.0:3004/v1/apis"
		headers := map[string]string{
			HeaderXForwardedFor:   "128.4.5.120",
			HeaderXForwardedHost:  "www.test.com",
			HeaderXForwardedProto: "proto",
			HeaderAccept:          "accept",
			HeaderAcceptEncoding:  "accept encoding",
			HeaderUserAgent:       "user agent",
			HeaderReferer:         "referrer",
			HeaderXLocation:       "continent=America,country=USA,city=LA,lat=34.0522 N,long=118.2437 W",
		}
		req, err := buildRequest(token, "GET", url, headers)
		assert.NoError(t, err)

		// Add claims to the request context
		clientId := uuid.New()
		sub := uuid.New()
		clientTenantId := uuid.New()
		claims := map[string]any{
			"client_id":        clientId.String(),
			"sub":              sub.String(),
			"client_tenant_id": clientTenantId.String(),
		}
		newReq := req.WithContext(jwtverifier.NewContext(req.Context(), claims))

		var eventsPublisher brokerclient.AuditEventsPublisher

		midl := auditMid(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			eventsPublisher = auditeventspublisher.FromContext(r.Context())
		}))

		w := &mockResponseWriter{}
		midl.ServeHTTP(w, newReq)

		assert.NotNil(t, publisher)
		assert.IsType(t, &auditeventspublisher.AuditEventsPublisher{}, eventsPublisher)

		// Call publish to capture the generated AuditLogEvent
		err = eventsPublisher.Publish(newReq.Context(), &testAuditEvent{})
		assert.NoError(t, err)

		auditLogEvent := publisher.auditEvent
		assert.Equal(t, 16, len(auditLogEvent.Attributes))

		for _, attr := range auditLogEvent.Attributes {
			switch attr.Key {
			case integrationevents.AttrKeyClientAccept:
				assert.Equal(t, "accept", attr.Value.(string))
			case integrationevents.AttrKeyClientAcceptEncoding:
				assert.Equal(t, "accept encoding", attr.Value.(string))
			case integrationevents.AttrKeyClientCity:
				assert.Equal(t, "LA", attr.Value.(string))
			case integrationevents.AttrKeyClientContinent:
				assert.Equal(t, "America", attr.Value.(string))
			case integrationevents.AttrKeyClientCountry:
				assert.Equal(t, "USA", attr.Value.(string))
			case integrationevents.AttrKeyClientHost:
				assert.Equal(t, "www.test.com", attr.Value.(string))
			case integrationevents.AttrKeyClientId:
				assert.Equal(t, clientId.String(), attr.Value.(string), " wrong AttrKeyClientId")
			case integrationevents.AttrKeyClientTenantId:
				assert.Equal(t, clientTenantId.String(), attr.Value.(string), " wrong AttrKeyClientTenanttId")
			case integrationevents.AttrKeyClientIp:
				assert.Equal(t, "128.4.5.120", attr.Value.(string))
			case integrationevents.AttrKeyClientLat:
				assert.Equal(t, "34.0522 N", attr.Value.(string))
			case integrationevents.AttrKeyClientLong:
				assert.Equal(t, "118.2437 W", attr.Value.(string))
			case integrationevents.AttrKeyClientProto:
				assert.Equal(t, "proto", attr.Value.(string))
			case integrationevents.AttrKeyClientRawUserAgent:
				assert.Equal(t, "user agent", attr.Value.(string))
			case integrationevents.AttrKeyClientReferer:
				assert.Equal(t, "referrer", attr.Value.(string))
			case integrationevents.AttrKeySourceName:
				assert.Equal(t, "httpAuditMiddlewareTest", attr.Value.(string))
			case integrationevents.AttrKeyUsertId:
				assert.Equal(t, sub.String(), attr.Value.(string), "wrong AttrKeyUsertId")
			}
		}
	})

	t.Run("should add events publisher into the context even when claims don't exist in context", func(t *testing.T) {
		publisher := &mockPublisher{}
		publisher.On("Publish", mock.Anything, mock.Anything).Return(nil)

		auditMid := HttpAuditMiddleware(publisher, "httpAuditMiddlewareTest")

		token := "eyJhbGciOiJSUzI1NiIsImtpZCI6IjI3MjlFRjY4MTYxQjFGQUQ1MkIzMTU2MjM4QkY2MUYxNzMwQjY5NzEiLCJ0eXAiOiJKV1QiLCJ4NXQiOiJKeW52YUJZYkg2MVNzeFZpT0w5aDhYTUxhWEUifQ.eyJuYmYiOjE2NTI3MzM1MDIsImV4cCI6MTY1MjczNzEwMiwiaXNzIjoiaHR0cHM6Ly9pZC1kZXYudmRtcy5pbyIsImF1ZCI6WyJpZC5hcGkiLCJodHRwczovL2lkLWRldi52ZG1zLmlvL3Jlc291cmNlcyJdLCJjbGllbnRfaWQiOiIwOWMxYzkxYS04NGJiLTRkYjQtODUyZC1iNDk2NzBjMmEzZDMiLCJqdGkiOiI4REI2OTRCQTEwOUZDOTRCOTQ2QTkwQkYzQjAyNTZERCIsImlhdCI6MTY1MjczMzUwMiwic2NvcGUiOlsiaWQuYXBpbS5hcGlzOnJlYWQiXX0.ZRxhhk0357HsihHQrqQBA4nIX4JlLJp3JwA4Pw9_ZJbBcORJmVZoTFfnSkcxBym2A1S_bGEBzS-Jo-MoY7YWkypy0YGVOk6bLehuPhk4r2NmYbIrxfLonIYWapJdKgLSNrhvOxoi6SJNObx_n2LOQgPmMbY_9CTnlL_jWsi12p0h2CITXmH7vDcDKo1Iy2goQrS0cLj_B9GJvQ1FiDMsl5G9tORA1D3VSoEUJ5hHRY4zl1Rjeu_jL7enwc5y1mPxk0wD5EflZQs7lG7z3-OHeIDLTr5VzHR5hTRXqBfiByGUi5VumZPHBDIMan7R5N2zBPel2dzYhhL94olCtrARTg"
		url := "http://0.0.0.0:3004/v1/apis"
		headers := map[string]string{
			HeaderXForwardedFor: "128.4.5.120",
		}
		req, err := buildRequest(token, "GET", url, headers)
		assert.NoError(t, err)

		var eventsPublisher brokerclient.AuditEventsPublisher

		midl := auditMid(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			eventsPublisher = auditeventspublisher.FromContext(r.Context())
		}))

		w := &mockResponseWriter{}
		midl.ServeHTTP(w, req)

		assert.NotNil(t, publisher)
		assert.IsType(t, &auditeventspublisher.AuditEventsPublisher{}, eventsPublisher)

		// Call publish to capture the generated AuditLogEvent
		err = eventsPublisher.Publish(req.Context(), &testAuditEvent{})
		assert.NoError(t, err)

		auditLogEvent := publisher.auditEvent
		assert.Equal(t, 2, len(auditLogEvent.Attributes))
	})
}
