package devices_test

import (
	"context"
	"fmt"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
	"github.com/hitoshiichikawa/apple-business-go/devices"
)

// 組織のデバイス一覧を取得する（ページングは自動）。
func ExampleService_List() {
	var c *applebusiness.Client // 実際には applebusiness.NewClient で生成する
	svc := devices.New(c)
	list, err := svc.List(context.Background(), nil)
	if err != nil {
		return
	}
	for _, d := range list {
		fmt.Println(d.ID, d.Attributes.SerialNumber, d.Attributes.Status)
	}
}

// デバイスを MDM サーバへ割り当て、完了までポーリングする。
// subStatus が COMPLETED_WITH_ERROR の場合は downloadUrl の CSV に明細がある。
func ExampleService_Assign() {
	var c *applebusiness.Client // 実際には applebusiness.NewClient で生成する
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
