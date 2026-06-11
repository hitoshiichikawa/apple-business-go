// Package auditevents covers the Audit Events category (read-only): /v1/auditEvents.
//
// Official model (confirmed against the DocC):
//
//	AuditEvent.attributes = the common envelope AuditEventCommonAttributes
//	  (eventDateTime / type / category / actor* / subject* / outcome / groupId / eventDataPropertyKey)
//	plus an event-specific payload (key name = eventDataPropertyKey, e.g. "eventDataDeviceAssignedToServer").
//
// The common fields are kept as typed values, while the event-specific payload is collected into
// EventData (raw JSON) and decoded into a concrete type via Payload().
package auditevents

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"time"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
)

// AuditEvent is an audit-event resource (type / id / attributes).
type AuditEvent = applebusiness.ResourceObject[Attributes]

// Attributes holds the audit-event attributes (the common envelope plus the event-specific payload).
type Attributes struct {
	EventDateTime        string `json:"eventDateTime,omitempty"`
	Type                 string `json:"type,omitempty"`     // AuditEventType
	Category             string `json:"category,omitempty"` // AuditEventCategory
	ActorType            string `json:"actorType,omitempty"`
	ActorID              string `json:"actorId,omitempty"`
	ActorName            string `json:"actorName,omitempty"`
	SubjectType          string `json:"subjectType,omitempty"`
	SubjectID            string `json:"subjectId,omitempty"`
	SubjectName          string `json:"subjectName,omitempty"`
	Outcome              string `json:"outcome,omitempty"` // AuditEventOutcome
	GroupID              string `json:"groupId,omitempty"`
	EventDataPropertyKey string `json:"eventDataPropertyKey,omitempty"`

	// EventData holds the raw JSON of the event-specific payload. Keys are "eventData..." (usually matching EventDataPropertyKey).
	EventData map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the common fields while collecting the "eventData..." keys into EventData.
func (a *Attributes) UnmarshalJSON(b []byte) error {
	type alias Attributes // メソッドを持たない別名で再帰を回避
	var base alias
	if err := json.Unmarshal(b, &base); err != nil {
		return err
	}
	*a = Attributes(base)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	for k, v := range raw {
		if strings.HasPrefix(k, "eventData") {
			if a.EventData == nil {
				a.EventData = make(map[string]json.RawMessage)
			}
			a.EventData[k] = v
		}
	}
	return nil
}

// Payload decodes the event-specific payload into v (using the key in EventDataPropertyKey).
func (a *Attributes) Payload(v any) error {
	key := a.EventDataPropertyKey
	if key == "" {
		for k := range a.EventData {
			key = k
			break
		}
	}
	raw, ok := a.EventData[key]
	if !ok {
		return nil
	}
	return json.Unmarshal(raw, v)
}

// Event-specific payload types (per the official DocC). Payloads with the same shape across
// multiple events are shared.
type (
	// DEVICE_ADDED_TO_ORG
	DeviceAddedToOrg struct {
		SerialNumber       string `json:"serialNumber,omitempty"`
		PurchaseSourceType string `json:"purchaseSourceType,omitempty"` // AuditEventPurchaseSourceType
		PurchaseSourceID   string `json:"purchaseSourceId,omitempty"`
	}
	// DEVICE_REMOVED_FROM_ORG
	DeviceRemovedFromOrg struct {
		SerialNumber      string `json:"serialNumber,omitempty"`
		ReleaseEntityID   string `json:"releaseEntityId,omitempty"`
		ReleaseEntityType string `json:"releaseEntityType,omitempty"` // AuditEventReleaseEntityType
	}
	// DEVICE_ASSIGNED_TO_SERVER
	DeviceAssignedToServer struct {
		SerialNumber     string `json:"serialNumber,omitempty"`
		TargetServerName string `json:"targetServerName,omitempty"`
	}
	// DEVICE_UNASSIGNED_FROM_SERVER
	DeviceUnassignedFromServer struct {
		SerialNumber string `json:"serialNumber,omitempty"`
	}
	// CONFIG_SETTINGS_CREATED / UPDATED / DELETED
	ConfigSettings struct {
		ConfigType    string `json:"configType,omitempty"`
		ConfigID      string `json:"configId,omitempty"`
		ConfigVersion string `json:"configVersion,omitempty"`
	}
	// COLLECTION_CREATED / UPDATED / DELETED
	Collection struct {
		Name        string `json:"name,omitempty"`
		Description string `json:"description,omitempty"`
	}
	// SUBSCRIPTION_CREATED / UPDATED / DELETED
	Subscription struct {
		PlanCaption string `json:"planCaption,omitempty"`
	}
	// SUBJECT_HAS_(ICLOUD_STORAGE|APPLECARE)_PURCHASE_ADDED / REMOVED
	PurchaseSubscription struct {
		SubscriptionID string `json:"subscriptionId,omitempty"`
	}
)

// AccountRoleLocation is a role/location pair (AuditEventAccountRoleLocation).
type AccountRoleLocation struct {
	RoleName                 string `json:"roleName,omitempty"`
	LocationUniqueIdentifier string `json:"locationUniqueIdentifier,omitempty"`
}

// Additional event-specific payloads (per the DocC).
type (
	// ACCOUNT_ROLE_LOCATION_CHANGED
	AccountRoleLocationChanged struct {
		AccountRoleLocationList []AccountRoleLocation `json:"accountRoleLocationList,omitempty"`
	}
	// API_ACCOUNT_ROLE_LOCATION_CHANGED
	APIAccountRoleLocationChanged struct {
		APIAccountRoleLocationList []AccountRoleLocation `json:"apiAccountRoleLocationList,omitempty"`
	}
	// API_ACCOUNT_CREATED_WITH_KEY / API_ACCOUNT_KEY_GENERATED / API_ACCOUNT_KEY_REVOKED
	APIAccountKey struct {
		KeyID string `json:"keyId,omitempty"`
	}
	// API_ACCOUNT_NAME_CHANGED
	APIAccountNameChanged struct {
		NewName string `json:"newName,omitempty"`
	}
)

// All AuditEventType values (33 in total). Compare against Attributes.Type to branch on the event.
// For the eventData payload of each type, see docs/apple-business-api-datatypes.md §4.3.
const (
	TypeDeviceAddedToOrg                       = "DEVICE_ADDED_TO_ORG"
	TypeDeviceRemovedFromOrg                   = "DEVICE_REMOVED_FROM_ORG"
	TypeDeviceAssignedToServer                 = "DEVICE_ASSIGNED_TO_SERVER"
	TypeDeviceUnassignedFromServer             = "DEVICE_UNASSIGNED_FROM_SERVER"
	TypeSubjectHasICloudStoragePurchaseAdded   = "SUBJECT_HAS_ICLOUD_STORAGE_PURCHASE_ADDED"
	TypeSubjectHasICloudStoragePurchaseRemoved = "SUBJECT_HAS_ICLOUD_STORAGE_PURCHASE_REMOVED"
	TypeSubjectHasAppleCarePurchaseAdded       = "SUBJECT_HAS_APPLECARE_PURCHASE_ADDED"
	TypeSubjectHasAppleCarePurchaseRemoved     = "SUBJECT_HAS_APPLECARE_PURCHASE_REMOVED"
	TypeDeviceIsErased                         = "DEVICE_IS_ERASED"
	TypeConfigSettingsCreated                  = "CONFIG_SETTINGS_CREATED"
	TypeConfigSettingsUpdated                  = "CONFIG_SETTINGS_UPDATED"
	TypeConfigSettingsDeleted                  = "CONFIG_SETTINGS_DELETED"
	TypeCollectionCreated                      = "COLLECTION_CREATED"
	TypeCollectionUpdated                      = "COLLECTION_UPDATED"
	TypeCollectionDeleted                      = "COLLECTION_DELETED"
	TypeSubscriptionCreated                    = "SUBSCRIPTION_CREATED"
	TypeSubscriptionUpdated                    = "SUBSCRIPTION_UPDATED"
	TypeSubscriptionDeleted                    = "SUBSCRIPTION_DELETED"
	TypeAccountRoleLocationChanged             = "ACCOUNT_ROLE_LOCATION_CHANGED"
	TypeAccountAdded                           = "ACCOUNT_ADDED"
	TypeAccountDeleted                         = "ACCOUNT_DELETED"
	TypeExternalAccountAssociated              = "EXTERNAL_ACCOUNT_ASSOCIATED"
	TypeExternalAccountDisassociated           = "EXTERNAL_ACCOUNT_DISASSOCIATED"
	TypeDomainAdded                            = "DOMAIN_ADDED"
	TypeDomainRemoved                          = "DOMAIN_REMOVED"
	TypeDomainVerified                         = "DOMAIN_VERIFIED"
	TypeAPIAccountCreatedWithKey               = "API_ACCOUNT_CREATED_WITH_KEY"
	TypeAPIAccountCreatedWithoutKey            = "API_ACCOUNT_CREATED_WITHOUT_KEY"
	TypeAPIAccountDeleted                      = "API_ACCOUNT_DELETED"
	TypeAPIAccountKeyRevoked                   = "API_ACCOUNT_KEY_REVOKED"
	TypeAPIAccountKeyGenerated                 = "API_ACCOUNT_KEY_GENERATED"
	TypeAPIAccountRoleLocationChanged          = "API_ACCOUNT_ROLE_LOCATION_CHANGED"
	TypeAPIAccountNameChanged                  = "API_ACCOUNT_NAME_CHANGED"
)

// Service exposes the audit-events endpoint. Construct with New.
type Service struct {
	c *applebusiness.Client
}

func New(c *applebusiness.Client) *Service { return &Service{c: c} }

// List retrieves audit events (all pages).
// Note: /v1/auditEvents requires filter[startTimestamp] (omitting it returns 400 PARAMETER_ERROR).
// ListRange is the simplest way to specify a time range. To pass it manually, set
// "filter[startTimestamp]" (required, ISO8601) and "filter[endTimestamp]" (optional, ISO8601) in q.
func (s *Service) List(ctx context.Context, q url.Values) ([]AuditEvent, error) {
	return applebusiness.List[Attributes](ctx, s.c, "/v1/auditEvents", q)
}

// ListRange retrieves audit events within a time range (all pages).
// start maps to filter[startTimestamp] (required); end maps to filter[endTimestamp] (omitted when zero).
// Both are sent as UTC RFC3339 (ISO8601). Additional filters can be passed via q
// (may be nil); the caller's q is not modified.
func (s *Service) ListRange(ctx context.Context, start, end time.Time, q url.Values) ([]AuditEvent, error) {
	merged := make(url.Values, len(q)+2)
	for k, v := range q {
		merged[k] = v
	}
	merged.Set("filter[startTimestamp]", start.UTC().Format(time.RFC3339))
	if !end.IsZero() {
		merged.Set("filter[endTimestamp]", end.UTC().Format(time.RFC3339))
	}
	return applebusiness.List[Attributes](ctx, s.c, "/v1/auditEvents", merged)
}
