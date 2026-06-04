package applebusiness_test

import (
	"context"
	"fmt"
	"os"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
)

// Basic creation of a NewClient. Functional options can set the User-Agent and more.
func ExampleNewClient() {
	pem, _ := os.ReadFile("abm_private_key.pem") // EC P-256 (.pem)
	c, err := applebusiness.NewClient(applebusiness.Config{
		BaseURL: applebusiness.DefaultBusinessBaseURL,
		Credentials: applebusiness.Credentials{
			ClientID:   "BUSINESSAPI.xxxxxxxx",
			TeamID:     "BUSINESSAPI.xxxxxxxx",
			KeyID:      "xxxxxxxx",
			PrivateKey: pem,
		},
	}, applebusiness.WithUserAgent("abm-scanner/1.0"), applebusiness.WithMaxRetries(5))
	if err != nil {
		return
	}
	_ = c
}

// Lazy paging with ListSeq (without loading everything into memory).
func ExampleListSeq() {
	var c *applebusiness.Client // in practice, create this with NewClient
	type deviceAttrs struct {
		SerialNumber string `json:"serialNumber"`
	}
	for d, err := range applebusiness.ListSeq[deviceAttrs](context.Background(), c, "/v1/orgDevices", nil) {
		if err != nil {
			break
		}
		fmt.Println(d.Attributes.SerialNumber)
	}
}

// Example of using the typed error predicates.
func ExampleIsNotFound() {
	var err error // in practice, the return value of Get, etc.
	switch {
	case applebusiness.IsNotFound(err):
		fmt.Println("not found")
	case applebusiness.IsRateLimited(err):
		fmt.Println("rate limited")
	case err != nil:
		fmt.Println("other error")
	}
}
