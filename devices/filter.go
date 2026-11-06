package devices

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
)

// Values of an organization device's product family, as accepted by
// filter[productFamily] (Apple Business API 2.7+ / Apple School Manager API
// 1.8+). They use the same spelling as DeviceAttributes.ProductFamily, which
// differs from the upper-case MdmProductFamily* values used by device
// management services.
const (
	ProductFamilyMac     = "Mac"
	ProductFamilyIPhone  = "iPhone"
	ProductFamilyIPad    = "iPad"
	ProductFamilyIPod    = "iPod"
	ProductFamilyAppleTV = "AppleTV"
	ProductFamilyWatch   = "Watch"
	ProductFamilyVision  = "Vision"
)

// Values of an organization device's purchase source type, as accepted by
// filter[purchaseSourceType].
const (
	PurchaseSourceTypeApple         = "APPLE"
	PurchaseSourceTypeReseller      = "RESELLER"
	PurchaseSourceTypeManuallyAdded = "MANUALLY_ADDED"
)

// Values of an organization device's assignment status
// (DeviceAttributes.Status), as accepted by filter[status].
const (
	DeviceStatusAssigned   = "ASSIGNED"
	DeviceStatusUnassigned = "UNASSIGNED"
)

// DeviceFilter narrows GET /v1/orgDevices (Apple Business API 2.7+ / Apple
// School Manager API 1.8+, added 2026-10-29). The zero value applies no
// filter.
//
// Each non-empty slice is sent as one comma-separated parameter, as the API
// documents. Empty strings are skipped, and a value must not itself contain a
// comma. Values are passed through unchanged, so values Apple adds later work
// without an SDK update. Use Query to build the url.Values for List,
// ListFiltered, or applebusiness.ListSeq.
type DeviceFilter struct {
	SerialNumbers       []string // filter[serialNumber]
	OrderNumbers        []string // filter[orderNumber]
	ProductFamilies     []string // filter[productFamily]: ProductFamily* values
	PurchaseSourceTypes []string // filter[purchaseSourceType]: PurchaseSourceType* values
	Statuses            []string // filter[status]: DeviceStatusAssigned / DeviceStatusUnassigned

	AddedToOrgDateTime TimeFilter // filter[addedToOrgDateTime] and its [GT]/[GTE]/[LT]/[LTE] forms
	OrderDateTime      TimeFilter // filter[orderDateTime] and its [GT]/[GTE]/[LT]/[LTE] forms
}

// TimeFilter filters on a date-time attribute of a device. Zero times are
// omitted. Times are sent in UTC as RFC 3339 (ISO 8601) with second
// precision, e.g. 2025-01-01T00:00:00Z.
type TimeFilter struct {
	// Equal is sent as filter[<attribute>] (comma-separated), which Apple
	// documents as filtering by the given date and time.
	Equal []time.Time
	GT    time.Time // filter[<attribute>][GT]: after
	GTE   time.Time // filter[<attribute>][GTE]: at or after
	LT    time.Time // filter[<attribute>][LT]: before
	LTE   time.Time // filter[<attribute>][LTE]: at or before
}

// Query returns a new url.Values holding base's parameters (for example limit
// or fields[orgDevices]) plus the filter's. base may be nil and is not
// modified. A non-empty filter field replaces the same-named parameter in base.
func (f DeviceFilter) Query(base url.Values) url.Values {
	q := make(url.Values, len(base)+7)
	for k, v := range base {
		q[k] = append([]string(nil), v...)
	}
	setFilterList(q, "filter[serialNumber]", f.SerialNumbers)
	setFilterList(q, "filter[orderNumber]", f.OrderNumbers)
	setFilterList(q, "filter[productFamily]", f.ProductFamilies)
	setFilterList(q, "filter[purchaseSourceType]", f.PurchaseSourceTypes)
	setFilterList(q, "filter[status]", f.Statuses)
	f.AddedToOrgDateTime.apply(q, "addedToOrgDateTime")
	f.OrderDateTime.apply(q, "orderDateTime")
	return q
}

// ListFiltered returns the organization devices matching f (pagination
// handled automatically). q holds other query parameters such as limit and
// may be nil; it is not modified. To iterate lazily instead, pass f.Query(q)
// to applebusiness.ListSeq[DeviceAttributes] with the path "/v1/orgDevices".
func (s *Service) ListFiltered(ctx context.Context, f DeviceFilter, q url.Values) ([]Device, error) {
	return applebusiness.List[DeviceAttributes](ctx, s.c, "/v1/orgDevices", f.Query(q))
}

// apply は filter[<attr>] と範囲指定 filter[<attr>][GT|GTE|LT|LTE] を q に設定する。
func (t TimeFilter) apply(q url.Values, attr string) {
	key := "filter[" + attr + "]"
	eq := make([]string, 0, len(t.Equal))
	for _, v := range t.Equal {
		if !v.IsZero() {
			eq = append(eq, filterTimeString(v))
		}
	}
	setFilterList(q, key, eq)
	for _, op := range []struct {
		suffix string
		v      time.Time
	}{{"[GT]", t.GT}, {"[GTE]", t.GTE}, {"[LT]", t.LT}, {"[LTE]", t.LTE}} {
		if !op.v.IsZero() {
			q.Set(key+op.suffix, filterTimeString(op.v))
		}
	}
}

// setFilterList は空文字列を除いた値をカンマ区切りで 1 つのパラメータに設定する。
// 値が 1 つも無ければ何もしない（base の同名パラメータは残る）。
func setFilterList(q url.Values, key string, vals []string) {
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		if v != "" {
			out = append(out, v)
		}
	}
	if len(out) > 0 {
		q.Set(key, strings.Join(out, ","))
	}
}

func filterTimeString(t time.Time) string { return t.UTC().Format(time.RFC3339) }
