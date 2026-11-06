// 受信の補助: 署名の検証と通知本文の解析。HTTP サーバそのものは持たない。

package webhooks

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
)

// SignatureHeader is the request header in which Apple sends a notification's
// signature (x-apple-signature on the wire). Its value is "hmacsha256=" followed
// by the hex HMAC-SHA256 of the request body, keyed with the webhook's secret.
const SignatureHeader = "X-Apple-Signature"

const signaturePrefix = "hmacsha256="

// DefaultMaxBodyBytes is the body size limit VerifyRequest uses when maxBytes
// is zero or negative. Notifications are a few hundred bytes.
const DefaultMaxBodyBytes int64 = 1 << 20

var (
	// ErrInvalidSignature is returned (possibly wrapped) when a signature is
	// missing, malformed, or does not match the body.
	ErrInvalidSignature = errors.New("webhooks: invalid signature")
	// ErrBodyTooLarge is returned by VerifyRequest when the body exceeds the limit.
	ErrBodyTooLarge = errors.New("webhooks: request body too large")
)

// VerifySignature checks that signature, the value of the x-apple-signature
// header ("hmacsha256=<hex>"), is the HMAC-SHA256 of body keyed with secret
// (the value given to Create or Update). The comparison runs in constant time.
// It returns nil when the signature is valid and an error wrapping
// ErrInvalidSignature otherwise.
//
// The signature covers the body only and carries no timestamp, so a valid
// notification can be replayed; drop duplicates by Notification.ID where it is
// present.
func VerifySignature(secret string, body []byte, signature string) error {
	sig := strings.TrimSpace(signature)
	if sig == "" {
		return fmt.Errorf("%w: header is empty", ErrInvalidSignature)
	}
	if len(sig) < len(signaturePrefix) || !strings.EqualFold(sig[:len(signaturePrefix)], signaturePrefix) {
		return fmt.Errorf("%w: missing %q prefix", ErrInvalidSignature, signaturePrefix)
	}
	got, err := hex.DecodeString(sig[len(signaturePrefix):])
	if err != nil {
		return fmt.Errorf("%w: digest is not hex", ErrInvalidSignature)
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	if !hmac.Equal(got, mac.Sum(nil)) {
		return fmt.Errorf("%w: digest does not match", ErrInvalidSignature)
	}
	return nil
}

// VerifyRequest reads the body of a notification request (at most maxBytes;
// zero or negative means DefaultMaxBodyBytes), verifies its x-apple-signature
// header with VerifySignature, and returns the body. r.Body is replaced so
// later handlers can read it again. It returns an error wrapping
// ErrBodyTooLarge or ErrInvalidSignature when the request is rejected.
func VerifyRequest(r *http.Request, secret string, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBodyBytes
	}
	if r.Body == nil {
		return nil, fmt.Errorf("%w: request has no body", ErrInvalidSignature)
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBytes+1))
	_ = r.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("webhooks: read request body: %w", err)
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("%w: more than %d bytes", ErrBodyTooLarge, maxBytes)
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	if err := VerifySignature(secret, body, r.Header.Get(SignatureHeader)); err != nil {
		return nil, err
	}
	return body, nil
}

// Values of Notification.Type: the event types as they appear in a
// notification body. They are spelled differently from the EventType* values
// used to subscribe (see Notification.EventType).
const (
	NotificationOrgDeviceAttributeChanged                  = "orgDeviceAttributeChanged"
	NotificationOrgDeviceAttributeMigrationDetailsChanged  = "orgDeviceAttributeMigrationDetailsChanged"
	NotificationOrgDeviceRelationshipAssignedServerChanged = "orgDeviceRelationshipAssignedServerChanged"
	NotificationOrgDeviceRelationshipWarrantyChanged       = "orgDeviceRelationshipWarrantyChanged"
	// NotificationWebhookPingCreated is the test notification sent by Ping.
	// It cannot be subscribed to.
	NotificationWebhookPingCreated = "webhookPingCreated"
)

// 通知本文の type（camelCase）→ 購読時の eventTypes（大文字のスネークケース）。
var notificationEventTypes = map[string]string{
	NotificationOrgDeviceAttributeChanged:                  EventTypeOrgDeviceAttributeChanged,
	NotificationOrgDeviceAttributeMigrationDetailsChanged:  EventTypeOrgDeviceAttributeMigrationDetailsChanged,
	NotificationOrgDeviceRelationshipAssignedServerChanged: EventTypeOrgDeviceRelationshipAssignedServerChanged,
	NotificationOrgDeviceRelationshipWarrantyChanged:       EventTypeOrgDeviceRelationshipWarrantyChanged,
}

// Notification is a webhook notification: the data object of the JSON body
// that Apple POSTs to a webhook's URL. Every event has the same shape; the
// event-specific values are in Attributes (decode them with DecodeAttributes).
// Use ID to drop duplicate deliveries when it is set; the ping payload
// returned by Deliveries had no id (observed 2026-11-06), so do not rely on it
// being present.
type Notification struct {
	Type          string                              `json:"type"`    // Notification* values
	ID            string                              `json:"id"`      // unique per event per the DocC; empty in the ping payload observed on 2026-11-06
	Version       int                                 `json:"version"` // version of the event's schema
	Attributes    json.RawMessage                     `json:"attributes,omitempty"`
	Relationships map[string]NotificationRelationship `json:"relationships,omitempty"`
}

// NotificationRelationship is one relationship of a notification, such as
// "instance" (the resource that changed) or "newAssignedServer".
type NotificationRelationship struct {
	Data  applebusiness.Data  `json:"data"`
	Links applebusiness.Links `json:"links,omitempty"`
}

// Attributes of the device events. The notification carries identifiers
// only; read the current state with the devices package.
type (
	// DeviceChanged is the attributes of orgDeviceAttributeChanged and
	// orgDeviceAttributeMigrationDetailsChanged.
	DeviceChanged struct {
		OrgDeviceSerialNumber string `json:"orgDeviceSerialNumber"`
	}
	// AssignedServerChanged is the attributes of
	// orgDeviceRelationshipAssignedServerChanged. The server is also linked as
	// the "newAssignedServer" relationship.
	AssignedServerChanged struct {
		OrgDeviceSerialNumber string `json:"orgDeviceSerialNumber"`
		NewMdmServerID        string `json:"newMdmServerId"`
	}
	// WarrantyChanged is the attributes of orgDeviceRelationshipWarrantyChanged.
	WarrantyChanged struct {
		OrgDeviceSerialNumber string `json:"orgDeviceSerialNumber"`
		AppleCareCoverageID   string `json:"appleCareCoverageId"`
	}
	// PingCreated is the attributes of webhookPingCreated.
	PingCreated struct {
		CreatedDateTime string `json:"createdDateTime"`
	}
)

// ParseNotification decodes a notification body ({"data":{...}}): the body of
// Apple's POST, or EventAttributes.Payload of a delivered event. Unknown event
// types are returned as is, with their raw attributes.
func ParseNotification(body []byte) (*Notification, error) {
	var doc struct {
		Data *Notification `json:"data"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("webhooks: parse notification: %w", err)
	}
	if doc.Data == nil || doc.Data.Type == "" {
		return nil, errors.New("webhooks: parse notification: no data.type")
	}
	return doc.Data, nil
}

// DecodeAttributes decodes the event-specific attributes into v, for example
// a *DeviceChanged for NotificationOrgDeviceAttributeChanged.
func (n *Notification) DecodeAttributes(v any) error {
	if len(n.Attributes) == 0 {
		return nil
	}
	return json.Unmarshal(n.Attributes, v)
}

// Instance returns the resource that changed (the "instance" relationship).
func (n *Notification) Instance() (applebusiness.Data, bool) {
	rel, ok := n.Relationships["instance"]
	return rel.Data, ok
}

// EventType returns the EventType* value that subscribes to this notification,
// or "" for the ping and for types this package does not know.
func (n *Notification) EventType() string { return notificationEventTypes[n.Type] }

// IsPing reports whether n is the test notification sent by Ping.
func (n *Notification) IsPing() bool { return n.Type == NotificationWebhookPingCreated }
