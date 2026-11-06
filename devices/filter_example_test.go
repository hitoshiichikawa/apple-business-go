package devices_test

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
	"github.com/hitoshiichikawa/apple-business-go/devices"
)

// List assigned iPhones and iPads added to the organization since 2025
// (Apple Business API 2.7+).
func ExampleService_ListFiltered() {
	var c *applebusiness.Client // in practice, create this with applebusiness.NewClient
	svc := devices.New(c)
	list, err := svc.ListFiltered(context.Background(), devices.DeviceFilter{
		ProductFamilies:    []string{devices.ProductFamilyIPhone, devices.ProductFamilyIPad},
		Statuses:           []string{devices.DeviceStatusAssigned},
		AddedToOrgDateTime: devices.TimeFilter{GTE: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
	}, nil)
	if err != nil {
		return
	}
	for _, d := range list {
		fmt.Println(d.Attributes.SerialNumber, d.Attributes.ProductFamily)
	}
}

// Iterate filtered devices page by page instead of loading them all at once.
func ExampleDeviceFilter_Query() {
	var c *applebusiness.Client // in practice, create this with applebusiness.NewClient
	q := devices.DeviceFilter{
		SerialNumbers: []string{"C02ABC123DEF", "DMPABC456GHI"},
	}.Query(url.Values{"limit": {"100"}})
	for d, err := range applebusiness.ListSeq[devices.DeviceAttributes](context.Background(), c, "/v1/orgDevices", q) {
		if err != nil {
			return
		}
		fmt.Println(d.ID, d.Attributes.Status)
	}
}
