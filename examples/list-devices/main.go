// Command list-devices lists organization devices.
//
// Usage:
//
//	export AXM_CLIENT_ID=BUSINESSAPI.xxxxxxxx-...
//	export AXM_TEAM_ID=$AXM_CLIENT_ID
//	export AXM_KEY_ID=xxxxxxxx-...
//	export AXM_PRIVATE_KEY_PATH=./abm_private_key.pem
//	go run ./examples/list-devices
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
	"github.com/hitoshiichikawa/apple-business-go/devices"
)

func main() {
	pem, err := os.ReadFile(requireEnv("AXM_PRIVATE_KEY_PATH"))
	if err != nil {
		log.Fatalf("read private key: %v", err)
	}

	c, err := applebusiness.NewClient(applebusiness.Config{
		BaseURL: envOrDefault("AXM_BASE_URL", applebusiness.DefaultBusinessBaseURL),
		Credentials: applebusiness.Credentials{
			ClientID:   requireEnv("AXM_CLIENT_ID"),
			TeamID:     requireEnv("AXM_TEAM_ID"),
			KeyID:      requireEnv("AXM_KEY_ID"),
			PrivateKey: pem,
			Scope:      envOrDefault("AXM_SCOPE", "business.api"),
		},
	})
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	list, err := devices.New(c).List(context.Background(), nil)
	if err != nil {
		log.Fatalf("list devices: %v", err)
	}

	fmt.Printf("%d devices\n", len(list))
	for _, d := range list {
		a := d.Attributes
		fmt.Printf("- %-20s %-14s %-10s %s\n", a.SerialNumber, a.ProductFamily, a.Status, d.ID)
	}
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
