package webhooks_test

import (
	"fmt"
	"net/http"

	"github.com/hitoshiichikawa/apple-business-go/webhooks"
)

// Verify a signature with the test vector from Apple's "Understanding webhook
// events" article.
func ExampleVerifySignature() {
	err := webhooks.VerifySignature("This is my secret", []byte("Hello, World!"),
		"hmacsha256=7f062172b01cb00b53ca068614674a3d982a34062a0f5d37687d5e3377e54657")
	fmt.Println(err)
	// Output: <nil>
}

// Receive notifications in your own HTTP server: verify the signature, parse
// the body, drop duplicates by ID, then re-read the device with the devices
// package (notifications carry identifiers only).
func ExampleParseNotification() {
	secret := "a-long-random-secret" // the value given to Create or Update
	mux := http.NewServeMux()
	mux.HandleFunc("POST /apple-business/webhook", func(w http.ResponseWriter, r *http.Request) {
		body, err := webhooks.VerifyRequest(r, secret, 0)
		if err != nil {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}
		n, err := webhooks.ParseNotification(body)
		if err != nil {
			http.Error(w, "bad notification", http.StatusBadRequest)
			return
		}
		switch n.Type {
		case webhooks.NotificationOrgDeviceAttributeChanged,
			webhooks.NotificationOrgDeviceAttributeMigrationDetailsChanged:
			var a webhooks.DeviceChanged
			if err := n.DecodeAttributes(&a); err == nil {
				fmt.Println("device changed:", a.OrgDeviceSerialNumber, n.ID)
			}
		case webhooks.NotificationOrgDeviceRelationshipAssignedServerChanged:
			var a webhooks.AssignedServerChanged
			if err := n.DecodeAttributes(&a); err == nil {
				fmt.Println("now assigned to:", a.NewMdmServerID)
			}
		case webhooks.NotificationWebhookPingCreated:
			// the test notification sent by Ping
		}
		w.WriteHeader(http.StatusNoContent)
	})
	_ = mux // in practice: http.ListenAndServeTLS(":443", cert, key, mux)
}
