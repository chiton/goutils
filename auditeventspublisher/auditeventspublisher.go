package auditeventspublisher

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gitlab.edgecastcdn.net/edgecast/web-platform/identity/brokerclient"
	"gitlab.edgecastcdn.net/edgecast/web-platform/identity/brokerclient/log"
	"gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/correlation"
	"gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/integrationevents"
)

type eventsPublisherContextKey int

const (
	key eventsPublisherContextKey = iota
)

type AuditEventsPublisher struct {
	publisher  brokerclient.Publisher
	attributes []integrationevents.Attribute
	source     string
}

var _ brokerclient.AuditEventsPublisher = &AuditEventsPublisher{}

func NewAuditEventsPublisher(publisher brokerclient.Publisher, attributes []integrationevents.Attribute, source string) *AuditEventsPublisher {
	return &AuditEventsPublisher{
		publisher:  publisher,
		attributes: attributes,
		source:     source,
	}
}

func (p *AuditEventsPublisher) Publish(ctx context.Context, event any) error {
	eventType := getEventType(event)

	auditLogEvent := &integrationevents.AuditLogEvent{
		OccuredOn:     time.Now().UTC(),
		Type:          fmt.Sprintf("%s.%s", p.source, strings.TrimLeft(eventType, "*")),
		CorrelationId: correlation.FromContext(ctx),
		Attributes:    p.attributes,
		Data:          event,
	}

	// Fill source
	auditLogEvent.Attributes = append(auditLogEvent.Attributes, integrationevents.Attribute{
		Key:   integrationevents.AttrKeySourceName,
		Value: p.source,
	})

	msg, err := newBrokerMessage(*auditLogEvent)
	if err != nil {
		return err
	}

	err = p.publisher.Publish(ctx, msg)
	if err != nil {
		return err
	}

	return nil
}

func getEventType(event any) string {
	parts := strings.Split(fmt.Sprintf("%T", event), ".")
	eventType := parts[0]

	len := len(parts)
	if len > 1 {
		eventType = parts[len-1]
	}

	return eventType
}

func newBrokerMessage(event integrationevents.AuditLogEvent) (*brokerclient.BrokerMessage, error) {
	m, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}

	return &brokerclient.BrokerMessage{
		Destination: integrationevents.TopicAudit,
		Headers: map[string][]byte{
			brokerclient.HeaderMessageType: []byte(event.Type),
		},
		Content: m,
	}, nil
}

// NewContext creates a new context enriched with the events publisher.
func NewContext(ctx context.Context, publisher brokerclient.AuditEventsPublisher) context.Context {
	return context.WithValue(ctx, key, publisher)
}

// FromContext returns the events publisher from the given context.
// If the publisher is not present in the context it will return a stub
// that all it does is to log a warning for every event that is published.
func FromContext(ctx context.Context) brokerclient.AuditEventsPublisher {
	if publisher, ok := ctx.Value(key).(brokerclient.AuditEventsPublisher); ok {
		return publisher
	}

	return &logPublisher{}
}

type logPublisher struct {
}

func (p *logPublisher) Publish(ctx context.Context, event any) error {
	log := log.FromContext(ctx)
	log.Warnw("Events publisher not found in context -- writing audit event to logs",
		"event", event,
	)

	return nil
}
