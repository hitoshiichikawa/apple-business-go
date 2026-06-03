// Command write-test は SDK の「書き込み」APIを一通り実行して動作確認する。
//
// ⚠️ 実組織に対する書き込みです。安全のため:
//   - 既定はドライラン（無変更）。実行は -yes が必須。
//   - Blueprint / Configuration は専用のテスト用リソースを「作成→更新→（任意でリレーション付替）→削除」する自己完結方式。
//     既定では最後に削除して後始末する（-keep で残す）。
//   - デバイス割り当て（Assign/Unassign）は実デバイスの状態を変えるため明示オプトイン。
//     -assign-server と -assign-device の両方を指定したときだけ実行し、元の割り当て先へベストエフォートで復元する。
//
// 認証情報は環境変数から（smoke-test / dump-all と同じ）:
//
//	AXM_CLIENT_ID, AXM_KEY_ID, AXM_PRIVATE_KEY_PATH（必須）
//	AXM_TEAM_ID, AXM_SCOPE, AXM_BASE_URL, AXM_TOKEN_URL（任意）
//
// 使い方:
//
//	go run ./examples/write-test                  # ドライラン（実行計画の表示のみ）
//	go run ./examples/write-test -yes             # Blueprint/Configuration の作成→更新→削除を実行
//	go run ./examples/write-test -yes -app <id>   # 併せて Blueprint の apps リレーション付替も試す
//	go run ./examples/write-test -yes -assign-server <mdmId> -assign-device <serial>  # 割り当ても試す（復元あり）
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
	"github.com/hitoshiichikawa/apple-business-go/blueprints"
	"github.com/hitoshiichikawa/apple-business-go/configurations"
	"github.com/hitoshiichikawa/apple-business-go/devices"
)

func main() {
	yes := flag.Bool("yes", false, "実際に書き込みを実行する（未指定はドライラン＝無変更）")
	keep := flag.Bool("keep", false, "作成したテスト用リソースを削除せず残す")
	appID := flag.String("app", "", "Blueprint の apps リレーション付替テストに使う app ID（任意）")
	assignServer := flag.String("assign-server", "", "割り当てテスト対象の MDM サーバ ID（-assign-device と併用で実行）")
	assignDevice := flag.String("assign-device", "", "割り当てテスト対象のデバイス ID(serial)")
	timeout := flag.Duration("timeout", 10*time.Minute, "全体タイムアウト")
	flag.Parse()

	pem, err := os.ReadFile(os.Getenv("AXM_PRIVATE_KEY_PATH"))
	if err != nil {
		fatalf("read private key (AXM_PRIVATE_KEY_PATH): %v", err)
	}
	if os.Getenv("AXM_CLIENT_ID") == "" || os.Getenv("AXM_KEY_ID") == "" {
		fatalf("AXM_CLIENT_ID and AXM_KEY_ID are required")
	}
	c, err := applebusiness.NewClient(applebusiness.Config{
		BaseURL: os.Getenv("AXM_BASE_URL"),
		Credentials: applebusiness.Credentials{
			ClientID:   os.Getenv("AXM_CLIENT_ID"),
			TeamID:     os.Getenv("AXM_TEAM_ID"),
			KeyID:      os.Getenv("AXM_KEY_ID"),
			PrivateKey: pem,
			Scope:      os.Getenv("AXM_SCOPE"),
		},
	}, applebusiness.WithUserAgent("apple-business-go/write-test"))
	if err != nil {
		fatalf("new client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// 認証確認（トークン取得）。ドライランでも実行して疎通を示す。
	tok, exp, terr := c.AccessToken()
	if terr != nil {
		fatalf("token exchange failed: %v", terr)
	}
	fmt.Printf("✓ token acquired: %s… (expires %s)\n\n", mask(tok), exp.Format(time.RFC3339))

	if !*yes {
		printPlan(*appID, *assignServer, *assignDevice, *keep)
		return
	}

	ok, fail := 0, 0
	report := func(label string, err error) {
		if err != nil {
			fail++
			fmt.Printf("✗ %s: %v\n", label, err)
			return
		}
		ok++
		fmt.Printf("✓ %s\n", label)
	}

	name := "abm-go-write-test-" + time.Now().Format("20060102-150405")
	fmt.Printf("テスト用リソース名: %s\n\n", name)

	// --- Blueprints: Create -> Update -> (rel) -> Delete ---
	fmt.Println("== Blueprints ==")
	bpSvc := blueprints.New(c)
	bp, err := bpSvc.Create(ctx, blueprints.CreateInput{Name: name, Description: "apple-business-go write-test"})
	report("blueprints.Create", err)
	if err == nil {
		_, uerr := bpSvc.Update(ctx, bp.ID, blueprints.UpdateInput{Name: strptr(name + " (updated)")})
		report("blueprints.Update", uerr)

		if *appID != "" {
			report("blueprints.AddTo(apps)", bpSvc.AddTo(ctx, bp.ID, blueprints.RelApps, []string{*appID}))
			report("blueprints.RemoveFrom(apps)", bpSvc.RemoveFrom(ctx, bp.ID, blueprints.RelApps, []string{*appID}))
		}

		if *keep {
			fmt.Printf("  （-keep のため Blueprint %s は残します）\n", bp.ID)
		} else {
			report("blueprints.Delete", bpSvc.Delete(ctx, bp.ID))
		}
	}

	// --- Configurations: Create -> Update -> Delete ---
	fmt.Println("\n== Configurations ==")
	cfgSvc := configurations.New(c)
	profile := base64.StdEncoding.EncodeToString([]byte(sampleMobileconfig(name)))
	cfg, err := cfgSvc.Create(ctx, configurations.CreateInput{
		Name:                   name,
		ConfiguredForPlatforms: []string{configurations.PlatformIOS},
		ConfigurationProfile:   profile, // byte=Base64
		Filename:               name + ".mobileconfig",
	})
	report("configurations.Create", err)
	if err == nil {
		_, uerr := cfgSvc.Update(ctx, cfg.ID, configurations.UpdateInput{Name: strptr(name + " (updated)")})
		report("configurations.Update", uerr)
		if *keep {
			fmt.Printf("  （-keep のため Configuration %s は残します）\n", cfg.ID)
		} else {
			report("configurations.Delete", cfgSvc.Delete(ctx, cfg.ID))
		}
	}

	// --- Devices: Assign/Unassign（オプトイン・復元あり） ---
	if *assignServer != "" && *assignDevice != "" {
		fmt.Println("\n== Devices (assign/unassign) ==")
		devSvc := devices.New(c)

		orig := ""
		if srv, e := devSvc.AssignedServer(ctx, *assignDevice); e == nil && srv != nil {
			orig = srv.ID
		}
		fmt.Printf("  対象 %s の元の割り当て先: %q\n", *assignDevice, orig)

		act, aerr := devSvc.Assign(ctx, *assignServer, []string{*assignDevice})
		report("devices.Assign", aerr)
		if aerr == nil {
			pollPrint(ctx, devSvc, act.ID)
		}

		// ベストエフォート復元: 元サーバがあれば戻し、無ければ解除。
		if orig != "" && orig != *assignServer {
			act2, rerr := devSvc.Assign(ctx, orig, []string{*assignDevice})
			report("devices.Assign(restore→"+orig+")", rerr)
			if rerr == nil {
				pollPrint(ctx, devSvc, act2.ID)
			}
		} else {
			act2, rerr := devSvc.Unassign(ctx, *assignServer, []string{*assignDevice})
			report("devices.Unassign", rerr)
			if rerr == nil {
				pollPrint(ctx, devSvc, act2.ID)
			}
		}
	}

	fmt.Printf("\n--- 完了: 成功 %d / 失敗 %d ---\n", ok, fail)
	if fail > 0 {
		os.Exit(1)
	}
}

func pollPrint(ctx context.Context, svc *devices.Service, activityID string) {
	final, err := svc.PollActivity(ctx, activityID, 3*time.Second)
	if err != nil {
		fmt.Printf("  poll %s: %v\n", activityID, err)
		return
	}
	fmt.Printf("  activity %s: status=%s subStatus=%s\n", activityID, final.Attributes.Status, final.Attributes.SubStatus)
}

func printPlan(appID, assignServer, assignDevice string, keep bool) {
	fmt.Println("=== DRY RUN（無変更）===")
	fmt.Println("-yes を付けると以下を実行します:")
	fmt.Println("  Blueprints:     Create → Update" + relNote(appID) + delNote(keep))
	fmt.Println("  Configurations: Create(CUSTOM_SETTING) → Update" + delNote(keep))
	if assignServer != "" && assignDevice != "" {
		fmt.Printf("  Devices:        Assign(%s→%s) → 復元(元へ戻す/解除)\n", assignDevice, assignServer)
	} else {
		fmt.Println("  Devices:        （-assign-server と -assign-device 指定時のみ。割り当ては実デバイスを変更します）")
	}
	fmt.Println("\nこれはドライランです（何も変更していません）。実際に実行するには -yes を付けて再実行してください。")
}

func relNote(appID string) string {
	if appID != "" {
		return " → AddTo(apps) → RemoveFrom(apps)"
	}
	return ""
}

func delNote(keep bool) string {
	if keep {
		return "（-keep: 削除しない）"
	}
	return " → Delete"
}

// sampleMobileconfig は検証用の最小 .mobileconfig（空の Configuration プロファイル）。
func sampleMobileconfig(name string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadType</key><string>Configuration</string>
  <key>PayloadVersion</key><integer>1</integer>
  <key>PayloadIdentifier</key><string>com.example.abmgo.` + name + `</string>
  <key>PayloadUUID</key><string>` + uuid() + `</string>
  <key>PayloadDisplayName</key><string>` + name + `</string>
  <key>PayloadContent</key><array/>
</dict>
</plist>`
}

func uuid() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func strptr(s string) *string { return &s }

func mask(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:8]
}

func fatalf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", a...)
	os.Exit(1)
}
