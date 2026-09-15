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

// Reuse access tokens per credential without keeping the private key in memory.
// fn runs only when a new token is needed (about once an hour), so the key can
// stay encrypted at rest and be decrypted just for that call.
func ExampleNewTokenSource() {
	decryptPrivateKey := func() ([]byte, error) {
		return os.ReadFile("abm_private_key.pem") // in practice, decrypt with your KMS / envelope key
	}

	// Create one source per credential (e.g. per tenant), cache it, and share it across Clients.
	ts := applebusiness.NewTokenSource(func() (applebusiness.Credentials, error) {
		pem, err := decryptPrivateKey()
		if err != nil {
			return applebusiness.Credentials{}, err
		}
		return applebusiness.Credentials{
			ClientID:   "BUSINESSAPI.xxxxxxxx",
			KeyID:      "xxxxxxxx",
			PrivateKey: pem,
		}, nil
	})

	c, err := applebusiness.NewClient(
		applebusiness.Config{BaseURL: applebusiness.DefaultBusinessBaseURL},
		applebusiness.WithTokenSource(ts),
	)
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
