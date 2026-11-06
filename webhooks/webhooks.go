// Package webhooks covers webhooks (Apple Business API 2.8+, added
// 2026-11-04): registering the HTTPS URL that Apple notifies when events occur
// in the organization, and reviewing, redelivering, and testing those
// notifications.
//
// This package calls the management API only. Receiving the notifications
// (the HTTP server that handles Apple's POSTs) belongs to the application.
package webhooks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
)

// JSON:API のリソース種別。
const (
	webhookType  = "webhooks"
	deliveryType = "webhookDeliveries"
	eventType    = "webhookEvents"
	pingType     = "webhookPings"
)

// Service exposes the webhook endpoints. Construct with New.
type Service struct {
	c *applebusiness.Client
}

func New(c *applebusiness.Client) *Service { return &Service{c: c} }

// List returns all webhooks (GET /v1/webhooks, pagination handled
// automatically).
func (s *Service) List(ctx context.Context, q url.Values) ([]Webhook, error) {
	return applebusiness.List[Attributes](ctx, s.c, "/v1/webhooks", q)
}

// Get returns a webhook by ID (GET /v1/webhooks/{id}).
func (s *Service) Get(ctx context.Context, id string) (*Webhook, error) {
	return applebusiness.Get[Attributes](ctx, s.c, "/v1/webhooks/"+url.PathEscape(id))
}

// CreateInput holds the parameters for creating a webhook. The API requires
// every field. Secret is the key Apple uses to sign each notification
// (HMAC-SHA256 of the body, sent in the x-apple-signature header). Choose a
// value that is hard to guess and store it yourself: the API never returns it.
type CreateInput struct {
	Name       string
	URL        string
	Secret     string
	EventTypes []string // EventType* values
	Enabled    bool
}

// Create registers a webhook (POST /v1/webhooks). Like other POSTs, it is not
// retried on 5xx or network errors; check with List before trying again.
func (s *Service) Create(ctx context.Context, in CreateInput) (*Webhook, error) {
	eventTypes := in.EventTypes
	if eventTypes == nil {
		eventTypes = []string{} // null ではなく空配列を送る
	}
	var body writeBody
	body.Data.Type = webhookType
	body.Data.Attributes = createAttrs{
		Name:       in.Name,
		Enabled:    in.Enabled,
		EventTypes: eventTypes,
		Secret:     in.Secret,
		URL:        in.URL,
	}
	return applebusiness.Create[Attributes](ctx, s.c, "/v1/webhooks", body)
}

// UpdateInput specifies only the attributes to change: a nil field (or a nil
// EventTypes) leaves the attribute unchanged. A new Secret takes effect for
// the following notifications, so update the receiver at the same time.
type UpdateInput struct {
	Name       *string
	URL        *string
	Secret     *string
	EventTypes []string // EventType* values
	Enabled    *bool
}

// Update changes the attributes of a webhook (PATCH /v1/webhooks/{id}).
func (s *Service) Update(ctx context.Context, id string, in UpdateInput) (*Webhook, error) {
	var body writeBody
	body.Data.Type = webhookType
	body.Data.ID = id
	body.Data.Attributes = updateAttrs(in)
	return applebusiness.Update[Attributes](ctx, s.c, "/v1/webhooks/"+url.PathEscape(id), body)
}

// Delete deletes a webhook (DELETE /v1/webhooks/{id}).
func (s *Service) Delete(ctx context.Context, id string) error {
	return applebusiness.Delete(ctx, s.c, "/v1/webhooks/"+url.PathEscape(id))
}

// DeliveryFilter narrows GET /v1/webhooks/{id}/deliveries beyond the required
// time range. The zero value adds nothing.
type DeliveryFilter struct {
	// States filters by delivery state (DeliveryState* values); several values
	// are sent comma-separated as filter[deliveryState].
	States []string
	// IncludeEvents also returns the events that the deliveries sent
	// (include=event); look them up with DeliveryList.Event.
	IncludeEvents bool
}

// DeliveryList is the result of Deliveries.
type DeliveryList struct {
	Deliveries []Delivery
	// Events holds the sent events by ID. It is filled only when
	// DeliveryFilter.IncludeEvents is set.
	Events map[string]Event
}

// Event returns the event that d sent, if the response included it.
func (l *DeliveryList) Event(d Delivery) (Event, bool) {
	e, ok := l.Events[d.EventID()]
	return e, ok
}

// Deliveries returns the deliveries of a webhook created at or after from and
// before to (GET /v1/webhooks/{id}/deliveries, pagination handled
// automatically). The API requires both ends of the range
// (filter[createdDateTimeAtOrAfter] and filter[createdDateTimeAtBefore]) even
// though the DocC lists them as optional, so from must be set; a zero to means
// now. Neither may be in the future: the API rejects that with 409
// ENTITY_ERROR.ATTRIBUTE.INVALID. Times are sent in UTC as RFC 3339 (whole
// seconds). q holds other query parameters such
// as limit and may be nil; it is not modified.
func (s *Service) Deliveries(ctx context.Context, webhookID string, from, to time.Time, f DeliveryFilter, q url.Values) (*DeliveryList, error) {
	if from.IsZero() {
		return nil, errors.New("webhooks: Deliveries requires a non-zero from (the API requires filter[createdDateTimeAtOrAfter])")
	}
	if to.IsZero() {
		to = time.Now()
	}
	v := f.query(q)
	v.Set("filter[createdDateTimeAtOrAfter]", from.UTC().Format(time.RFC3339))
	v.Set("filter[createdDateTimeAtBefore]", to.UTC().Format(time.RFC3339))
	endpoint := s.c.BaseURL() + "/v1/webhooks/" + url.PathEscape(webhookID) + "/deliveries?" + v.Encode()
	out := &DeliveryList{Events: map[string]Event{}}
	// コアの List は included を返さないため、links.next を自前で辿る。
	for endpoint != "" {
		var page deliveriesPage
		if err := s.c.Do(ctx, http.MethodGet, endpoint, nil, &page); err != nil {
			return nil, err
		}
		out.Deliveries = append(out.Deliveries, page.Data...)
		for _, e := range page.Included {
			if e.Type == eventType {
				out.Events[e.ID] = e
			}
		}
		endpoint = page.Links.Next
	}
	return out, nil
}

// Redeliver asks Apple to send a past delivery again
// (POST /v1/webhookDeliveries), for example after fixing the receiver. It
// returns the new delivery, whose Redelivery attribute is true.
func (s *Service) Redeliver(ctx context.Context, deliveryID string) (*Delivery, error) {
	var body relationshipBody
	body.Data.Type = deliveryType
	body.Data.Relationships = map[string]relationship{
		"template": {Data: applebusiness.Data{Type: deliveryType, ID: deliveryID}},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("webhooks: marshal body: %w", err)
	}
	// 応答の data は relationships を持つため、コアの Create（ResourceObject）ではなく Do で受ける。
	var resp struct {
		Data Delivery `json:"data"`
	}
	if err := s.c.Do(ctx, http.MethodPost, s.c.BaseURL()+"/v1/webhookDeliveries", raw, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// Ping asks Apple to send a test notification to the webhook's URL
// (POST /v1/webhookPings) and returns the ping's ID. To see whether it
// arrived, check the webhook's deliveries with Deliveries.
func (s *Service) Ping(ctx context.Context, webhookID string) (string, error) {
	var body relationshipBody
	body.Data.Type = pingType
	body.Data.Relationships = map[string]relationship{
		"webhook": {Data: applebusiness.Data{Type: webhookType, ID: webhookID}},
	}
	ping, err := applebusiness.Create[struct{}](ctx, s.c, "/v1/webhookPings", body)
	if err != nil {
		return "", err
	}
	return ping.ID, nil
}

// query は base の写しに配信一覧のフィルタを設定して返す（base は変更しない）。
func (f DeliveryFilter) query(base url.Values) url.Values {
	q := make(url.Values, len(base)+4)
	for k, v := range base {
		q[k] = append([]string(nil), v...)
	}
	states := make([]string, 0, len(f.States))
	for _, st := range f.States {
		if st != "" {
			states = append(states, st)
		}
	}
	if len(states) > 0 {
		q.Set("filter[deliveryState]", strings.Join(states, ","))
	}
	if f.IncludeEvents {
		q.Set("include", "event")
	}
	return q
}

// --- リクエスト / レスポンスの内部型 ------------------------------------------

type writeBody struct {
	Data struct {
		Type       string `json:"type"`
		ID         string `json:"id,omitempty"`
		Attributes any    `json:"attributes"`
	} `json:"data"`
}

// 作成時はすべて必須なので omitempty を付けない（enabled=false も送る）。
type createAttrs struct {
	Name       string   `json:"name"`
	Enabled    bool     `json:"enabled"`
	EventTypes []string `json:"eventTypes"`
	Secret     string   `json:"secret"`
	URL        string   `json:"url"`
}

type updateAttrs struct {
	Name       *string  `json:"name,omitempty"`
	URL        *string  `json:"url,omitempty"`
	Secret     *string  `json:"secret,omitempty"`
	EventTypes []string `json:"eventTypes,omitempty"`
	Enabled    *bool    `json:"enabled,omitempty"`
}

type relationship struct {
	Data applebusiness.Data `json:"data"`
}

type relationshipBody struct {
	Data struct {
		Type          string                  `json:"type"`
		Relationships map[string]relationship `json:"relationships"`
	} `json:"data"`
}

type deliveriesPage struct {
	Data     []Delivery          `json:"data"`
	Included []Event             `json:"included"`
	Links    applebusiness.Links `json:"links"`
}
