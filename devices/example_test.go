package devices_test

import (
	"context"
	"fmt"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
	"github.com/hitoshiichikawa/apple-business-go/devices"
)

// List the organization's devices (pagination is automatic).
func ExampleService_List() {
	var c *applebusiness.Client // in practice, create this with applebusiness.NewClient
	svc := devices.New(c)
	list, err := svc.List(context.Background(), nil)
	if err != nil {
		return
	}
	for _, d := range list {
		fmt.Println(d.ID, d.Attributes.SerialNumber, d.Attributes.Status)
	}
}

// Assign devices to an MDM server and poll until completion.
// If subStatus is COMPLETED_WITH_ERROR, the CSV at downloadUrl has the details.
func ExampleService_Assign() {
	var c *applebusiness.Client // in practice, create this with applebusiness.NewClient
	svc := devices.New(c)
	act, err := svc.Assign(context.Background(), "MDM_SERVER_ID", []string{"DEVICE_ID"})
	if err != nil {
		return
	}
	final, err := svc.PollActivity(context.Background(), act.ID, 0)
	if err != nil {
		return
	}
	fmt.Println(final.Attributes.Status, final.Attributes.SubStatus)
}
