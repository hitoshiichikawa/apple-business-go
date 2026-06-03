// Package auditevents covers the Audit Events category (read-only): /v1/auditEvents.
//
// 公式モデル（DocC 確認済み）:
//
//	AuditEvent.attributes = 共通エンベロープ AuditEventCommonAttributes
//	  （eventDateTime / type / category / actor* / subject* / outcome / groupId / eventDataPropertyKey）
//	＋ イベント固有ペイロード（キー名 = eventDataPropertyKey、例 "eventDataDeviceAssignedToServer"）。
//
// 共通項目は型付きで保持し、イベント固有ペイロードは EventData(生JSON) に収集して
// Payload() で個別型へデコードする。
package auditevents

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"time"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
)

// AuditEvent は監査イベントリソース（type / id / attributes）。
type AuditEvent = applebusiness.ResourceObject[Attributes]

// Attributes は監査イベントの属性（共通エンベロープ + イベント固有ペイロード）。
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

	// EventData はイベント固有ペイロードの生JSON。キーは "eventData..."（通常 EventDataPropertyKey と一致）。
	EventData map[string]json.RawMessage `json:"-"`
}

// UnmarshalJSON は共通フィールドをデコードしつつ、"eventData..." キーを EventData に収集する。
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

// Payload はイベント固有ペイロードを v にデコードする（EventDataPropertyKey のキーを使用）。
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

// イベント固有ペイロードの型（公式 DocC 準拠）。複数イベントで形が同じものは共通化。
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

// AccountRoleLocation はロール/ロケーションの組（AuditEventAccountRoleLocation）。
type AccountRoleLocation struct {
	RoleName                 string `json:"roleName,omitempty"`
	LocationUniqueIdentifier string `json:"locationUniqueIdentifier,omitempty"`
}

// 追加のイベント固有ペイロード（DocC 準拠）。
type (
	// ACCOUNT_ROLE_LOCATION_CHANGED
	AccountRoleLocationChanged struct {
		AccountRoleLocationList []AccountRoleLocation `json:"accountRoleLocationList,omitempty"`
	}
	// API_ACCOUNT_ROLE_LOCATION_CHANGED
	ApiAccountRoleLocationChanged struct {
		ApiAccountRoleLocationList []AccountRoleLocation `json:"apiAccountRoleLocationList,omitempty"`
	}
	// API_ACCOUNT_CREATED_WITH_KEY / API_ACCOUNT_KEY_GENERATED / API_ACCOUNT_KEY_REVOKED
	ApiAccountKey struct {
		KeyID string `json:"keyId,omitempty"`
	}
	// API_ACCOUNT_NAME_CHANGED
	ApiAccountNameChanged struct {
		NewName string `json:"newName,omitempty"`
	}
)

// AuditEventType の全値（33種）。Attributes.Type と比較して分岐に使う。
// type ごとの eventData ペイロードは docs/apple-business-api-datatypes.md §4.3 を参照。
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
	TypeApiAccountCreatedWithKey               = "API_ACCOUNT_CREATED_WITH_KEY"
	TypeApiAccountCreatedWithoutKey            = "API_ACCOUNT_CREATED_WITHOUT_KEY"
	TypeApiAccountDeleted                      = "API_ACCOUNT_DELETED"
	TypeApiAccountKeyRevoked                   = "API_ACCOUNT_KEY_REVOKED"
	TypeApiAccountKeyGenerated                 = "API_ACCOUNT_KEY_GENERATED"
	TypeApiAccountRoleLocationChanged          = "API_ACCOUNT_ROLE_LOCATION_CHANGED"
	TypeApiAccountNameChanged                  = "API_ACCOUNT_NAME_CHANGED"
)

// Service exposes the audit-events endpoint. Construct with New.
type Service struct {
	c *applebusiness.Client
}

func New(c *applebusiness.Client) *Service { return &Service{c: c} }

// List は監査イベントを取得する（全ページ）。
// 注意: /v1/auditEvents は filter[startTimestamp] が必須（未指定だと 400 PARAMETER_ERROR）。
// 期間指定には ListRange を使うのが簡単。手動で渡す場合は q に
// "filter[startTimestamp]"（必須, ISO8601）と "filter[endTimestamp]"（任意, ISO8601）を設定する。
func (s *Service) List(ctx context.Context, q url.Values) ([]AuditEvent, error) {
	return applebusiness.List[Attributes](ctx, s.c, "/v1/auditEvents", q)
}

// ListRange は時間範囲を指定して監査イベントを取得する（全ページ）。
// start は filter[startTimestamp]（必須）、end は filter[endTimestamp]（ゼロ値なら付与しない）。
// いずれも UTC の RFC3339(ISO8601) で送る。追加の絞り込みは q で渡す（nil可）。
func (s *Service) ListRange(ctx context.Context, start, end time.Time, q url.Values) ([]AuditEvent, error) {
	if q == nil {
		q = url.Values{}
	}
	q.Set("filter[startTimestamp]", start.UTC().Format(time.RFC3339))
	if !end.IsZero() {
		q.Set("filter[endTimestamp]", end.UTC().Format(time.RFC3339))
	}
	return applebusiness.List[Attributes](ctx, s.c, "/v1/auditEvents", q)
}
