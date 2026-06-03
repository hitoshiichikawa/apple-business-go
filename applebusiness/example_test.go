package applebusiness_test

import (
	"context"
	"fmt"
	"os"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
)

// NewClient の基本的な生成。Functional Options で User-Agent などを指定できる。
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

// ListSeq による遅延ページング（全件をメモリに載せない）。
func ExampleListSeq() {
	var c *applebusiness.Client // 実際には NewClient で生成する
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

// 型付きエラー判定の利用例。
func ExampleIsNotFound() {
	var err error // 実際には Get などの戻り値
	switch {
	case applebusiness.IsNotFound(err):
		fmt.Println("not found")
	case applebusiness.IsRateLimited(err):
		fmt.Println("rate limited")
	case err != nil:
		fmt.Println("other error")
	}
}
