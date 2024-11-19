package integrationevents

import (
	"time"
)

const (
	AttrKeyUsertId = "u.id"

	AttrKeySourceId   = "src.id"
	AttrKeySourceName = "src.name"
	AttrKeySourceType = "src.type"

	AttrKeyClientId             = "c.id"
	AttrKeyClientTenantId       = "c.tenantid"
	AttrKeyClientIp             = "c.ip"
	AttrKeyClientReferer        = "c.referer"
	AttrKeyClientProto          = "c.proto"
	AttrKeyClientHost           = "c.host"
	AttrKeyClientAccept         = "c.accept"
	AttrKeyClientAcceptEncoding = "c.acceptencoding"
	AttrKeyClientContinent      = "c.continent"
	AttrKeyClientCountry        = "c.country"
	AttrKeyClientCity           = "c.city"
	AttrKeyClientLat            = "c.lat"
	AttrKeyClientLong           = "c.long"
	AttrKeyClientOS             = "c.os"
	AttrKeyClientBrowser        = "c.browser"
	AttrKeyClientRawUserAgent   = "c.rawua"
)

type Attribute struct {
	Key   string `json:"k"`
	Value any    `json:"v"`
}

type AuditLogEvent struct {
	OccuredOn     time.Time   `json:"on"`
	Type          string      `json:"type"`
	CorrelationId string      `json:"cid"`
	Attributes    []Attribute `json:"attrs"`
	Data          any         `json:"d"`
}
