package webhooks_test

import (
	"context"
	"fmt"
	"time"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
	"github.com/hitoshiichikawa/apple-business-go/webhooks"
)

// Register a webhook for device attribute changes (Apple Business API 2.8+).
// Keep the secret: Apple signs each notification with it, and the API never
// returns it.
func ExampleService_Create() {
	var c *applebusiness.Client // in practice, create this with applebusiness.NewClient
	svc := webhooks.New(c)
	wh, err := svc.Create(context.Background(), webhooks.CreateInput{
		Name:       "device-sync",
		URL:        "https://example.com/apple-business/webhook",
		Secret:     "a-long-random-secret",
		EventTypes: []string{webhooks.EventTypeOrgDeviceAttributeChanged},
		Enabled:    true,
	})
	if err != nil {
		return
	}
	fmt.Println(wh.ID)
}

// Send a test ping, then review the failed deliveries of the last day together
// with the events they carried.
func ExampleService_Deliveries() {
	var c *applebusiness.Client // in practice, create this with applebusiness.NewClient
	svc := webhooks.New(c)
	ctx := context.Background()
	if _, err := svc.Ping(ctx, "WEBHOOK_ID"); err != nil {
		return
	}
	list, err := svc.Deliveries(ctx, "WEBHOOK_ID", webhooks.DeliveryFilter{
		States:           []string{webhooks.DeliveryStateFailed},
		CreatedAtOrAfter: time.Now().Add(-24 * time.Hour),
		IncludeEvents:    true,
	}, nil)
	if err != nil {
		return
	}
	for _, d := range list.Deliveries {
		ev, _ := list.Event(d)
		fmt.Println(d.ID, d.Attributes.ErrorMessage, ev.Attributes.IsPingEvent)
		// after fixing the receiver: svc.Redeliver(ctx, d.ID)
	}
}
