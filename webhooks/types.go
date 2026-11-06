package webhooks

import "github.com/hitoshiichikawa/apple-business-go/applebusiness"

// Webhook is a webhook resource (type "webhooks").
type Webhook = applebusiness.ResourceObject[Attributes]

// Attributes holds the attributes of a webhook. The secret is write-only: the
// API never returns it.
type Attributes struct {
	Name            string   `json:"name,omitempty"`
	Enabled         bool     `json:"enabled"`
	EventTypes      []string `json:"eventTypes,omitempty"` // EventType* values
	URL             string   `json:"url,omitempty"`
	CreatedDateTime string   `json:"createdDateTime,omitempty"`
	UpdatedDateTime string   `json:"updatedDateTime,omitempty"`
}

// Values of WebhookEventType: the event types a webhook can subscribe to
// (Attributes.EventTypes). The test ping event cannot be subscribed to and is
// not listed.
const (
	EventTypeOrgDeviceAttributeChanged                  = "ORG_DEVICE_ATTRIBUTE_CHANGED"
	EventTypeOrgDeviceAttributeMigrationDetailsChanged  = "ORG_DEVICE_ATTRIBUTE_MIGRATION_DETAILS_CHANGED"
	EventTypeOrgDeviceRelationshipAssignedServerChanged = "ORG_DEVICE_RELATIONSHIP_ASSIGNED_SERVER_CHANGED"
	EventTypeOrgDeviceRelationshipWarrantyChanged       = "ORG_DEVICE_RELATIONSHIP_WARRANTY_CHANGED"
)

// Values of a delivery's state (DeliveryAttributes.DeliveryState).
const (
	DeliveryStateSucceeded = "SUCCEEDED"
	DeliveryStateFailed    = "FAILED"
	DeliveryStatePending   = "PENDING"
)

// Delivery is one notification that Apple sent, or tried to send, to a
// webhook's URL (type "webhookDeliveries").
type Delivery struct {
	Type          string                `json:"type"`
	ID            string                `json:"id"`
	Attributes    DeliveryAttributes    `json:"attributes"`
	Relationships DeliveryRelationships `json:"relationships"`
	Links         applebusiness.Links   `json:"links,omitempty"`
}

// EventID returns the ID of the event that the delivery sent, or "" when the
// response does not carry the relationship.
func (d Delivery) EventID() string {
	if d.Relationships.Event.Data == nil {
		return ""
	}
	return d.Relationships.Event.Data.ID
}

// DeliveryAttributes holds the attributes of a delivery. ErrorMessage
// describes the error when the webhook URL returned no HTTP response (for
// example, the host couldn't be resolved); otherwise Response describes what
// it returned.
type DeliveryAttributes struct {
	CreatedDateTime string            `json:"createdDateTime,omitempty"`
	DeliveryState   string            `json:"deliveryState,omitempty"` // DeliveryState* values
	ErrorMessage    string            `json:"errorMessage,omitempty"`
	Redelivery      bool              `json:"redelivery,omitempty"`
	SentDateTime    string            `json:"sentDateTime,omitempty"`
	Request         *DeliveryRequest  `json:"request,omitempty"`
	Response        *DeliveryResponse `json:"response,omitempty"`
}

// DeliveryRequest describes the request that Apple sent.
type DeliveryRequest struct {
	URL string `json:"url,omitempty"`
}

// DeliveryResponse describes how the webhook URL responded. Body is present
// only when the status code is not 2xx.
type DeliveryResponse struct {
	HTTPStatusCode int    `json:"httpStatusCode,omitempty"`
	Body           string `json:"body,omitempty"`
}

// DeliveryRelationships holds the relationships of a delivery.
type DeliveryRelationships struct {
	Event DeliveryEventRelationship `json:"event"`
}

// DeliveryEventRelationship links a delivery to the event it sent.
type DeliveryEventRelationship struct {
	Data  *applebusiness.Data `json:"data,omitempty"`
	Links applebusiness.Links `json:"links,omitempty"`
}

// Event is a webhook event (type "webhookEvents"). Deliveries returns it when
// DeliveryFilter.IncludeEvents is set.
type Event = applebusiness.ResourceObject[EventAttributes]

// EventAttributes holds the attributes of a webhook event. Payload is the
// notification body that was sent, as a JSON string.
type EventAttributes struct {
	EventType       string `json:"eventType,omitempty"` // EventType* values
	Payload         string `json:"payload,omitempty"`
	IsPingEvent     bool   `json:"isPingEvent,omitempty"`
	CreatedDateTime string `json:"createdDateTime,omitempty"`
}
