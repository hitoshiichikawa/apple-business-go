// Command write-test は SDK の「書き込み」APIを一通り実行して動作確認する。
//
// ⚠️ 実組織に対する書き込みです。安全のため:
//   - 既定はドライラン（無変更）。実行は -yes が必須。
//   - Blueprint / Configuration は専用のテスト用リソースを「作成→更新→（任意でリレーション付替）→削除」する自己完結方式。
//     既定では最後に削除して後始末する（-keep で残す）。
//   - MDM サーバ（API 2.1 の CRUD）も同じ自己完結方式。自己署名 X.509 証明書をその場で生成して
//     「作成→取得→更新→削除」する。デバイスは一切割り当てないため削除で完全に消える。
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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"flag"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"time"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
	"github.com/hitoshiichikawa/apple-business-go/blueprints"
	"github.com/hitoshiichikawa/apple-business-go/configurations"
	"github.com/hitoshiichikawa/apple-business-go/devices"
	"github.com/hitoshiichikawa/apple-business-go/people"
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

	// --- Configurations: Create -> Update（削除は Blueprint の後で後始末）---
	// configurationProfile は .mobileconfig の中身（生 XML）をそのまま渡す。
	// 実機では Base64 化すると 400（plist type mismatch）になる（reference.md §7.3 と一致）。
	// この Configuration は直後の Blueprint の「中身」にも流用する（無害な Web クリップ）。
	fmt.Println("== Configurations ==")
	cfgSvc := configurations.New(c)
	cfg, cerr := cfgSvc.Create(ctx, configurations.CreateInput{
		Name:                   name,
		ConfiguredForPlatforms: []string{configurations.PlatformIOS},
		ConfigurationProfile:   sampleMobileconfig(name),
		Filename:               name + ".mobileconfig",
	})
	report("configurations.Create", cerr)
	if cerr == nil {
		_, uerr := cfgSvc.Update(ctx, cfg.ID, configurations.UpdateInput{Name: strptr(name + " (updated)")})
		report("configurations.Update", uerr)
	}

	// --- Blueprints: Create -> Update -> (rel) -> Delete ---
	// 実機では作成時に「中身(apps/packages/configurations)」と「割り当て先(orgDevices/users/userGroups)」の
	// 両カテゴリが最低1つずつ必須（reference.md の「任意」は誤り。配信なしでは作成できない）。
	// 中身は上で作った無害な Web クリップ Configuration を使い、割り当て先は userGroups→users の順で1件だけ
	// 一時的に付ける（実デバイスへの直接割り当ては避ける）。Blueprint は最後に削除して後始末する。
	fmt.Println("\n== Blueprints ==")
	bpSvc := blueprints.New(c)
	if cerr != nil {
		fmt.Println("⚠ Configuration 作成に失敗したため Blueprint の中身を用意できません。Blueprints をスキップします。")
	} else if bpIn, targetLabel, hasTarget := pickBlueprintTarget(ctx, c); !hasTarget {
		fmt.Println("⚠ users/userGroups が1件も無く、Blueprint 作成に必須の割り当て先を用意できません。Blueprints をスキップします。")
	} else {
		bpIn.Name = name
		bpIn.Description = "apple-business-go write-test"
		bpIn.Configurations = []string{cfg.ID} // 中身＝無害な Web クリップ Configuration
		fmt.Printf("  中身: configurations:%s（無害な Web クリップ）／ 割り当て先(一時): %s\n", cfg.ID, targetLabel)
		bp, berr := bpSvc.Create(ctx, bpIn)
		report("blueprints.Create", berr)
		if berr == nil {
			// Blueprint 名は英数字・ハイフン等に限られ、スペースや括弧は 409 ENTITY_ERROR.ATTRIBUTE.INVALID になる
			// （Configuration 名はスペース・括弧可。リソースごとに名前制約が異なる）。
			_, uerr := bpSvc.Update(ctx, bp.ID, blueprints.UpdateInput{Name: strptr(name + "-updated")})
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
	}

	// --- Configuration の後始末（Blueprint が参照し終わった後に削除）---
	if cerr == nil {
		if *keep {
			fmt.Printf("  （-keep のため Configuration %s は残します）\n", cfg.ID)
		} else {
			report("configurations.Delete", cfgSvc.Delete(ctx, cfg.ID))
		}
	}

	// --- MdmServers: Create -> Get -> Update -> Delete（API 2.1、自己完結）---
	// 証明書は自己署名（その場で生成、鍵は破棄）。デバイスは割り当てないため
	// deviceCount=0 のまま削除できる。証明書 data の形式は DocC どおり Base64 の DER。
	fmt.Println("\n== MdmServers (API 2.1) ==")
	devSvc := devices.New(c)
	srv, serr := devSvc.CreateMdmServer(ctx, devices.CreateMdmServerInput{
		ServerName:        name,
		ServerCertificate: devices.MdmServerCertificate{Name: name + ".cer", Data: selfSignedCertBase64(name)},
	})
	report("devices.CreateMdmServer", serr)
	if serr == nil {
		got, gerr := devSvc.GetMdmServer(ctx, srv.ID)
		report("devices.GetMdmServer", gerr)
		if gerr == nil {
			fmt.Printf("  id=%s status=%s deviceCount=%d\n", got.ID, got.Attributes.Status, got.Attributes.DeviceCount)
		}

		_, uerr := devSvc.UpdateMdmServer(ctx, srv.ID, devices.UpdateMdmServerInput{
			ServerName:             strptr(name + "-updated"),
			DefaultProductFamilies: []string{devices.MdmProductFamilyIPhone, devices.MdmProductFamilyIPad},
		})
		report("devices.UpdateMdmServer", uerr)

		if *keep {
			fmt.Printf("  （-keep のため MDM サーバ %s は残します）\n", srv.ID)
		} else {
			report("devices.DeleteMdmServer", devSvc.DeleteMdmServer(ctx, srv.ID))
		}
	}

	// --- Devices: Assign/Unassign（オプトイン・復元あり） ---
	if *assignServer != "" && *assignDevice != "" {
		fmt.Println("\n== Devices (assign/unassign) ==")

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

// pickBlueprintTarget は Blueprint の割り当て先を1件だけ探す（userGroups→users の順）。
// 実機では作成時に割り当て先(orgDevices/users/userGroups)が必須。実デバイスへの直接割り当ては
// 避けるため orgDevices は使わない。全件は辿らず ListSeq で最初の1件だけ取得する（limit=1）。
func pickBlueprintTarget(ctx context.Context, c *applebusiness.Client) (blueprints.CreateInput, string, bool) {
	q := url.Values{"limit": {"1"}}
	for g, err := range applebusiness.ListSeq[people.UserGroupAttributes](ctx, c, "/v1/userGroups", q) {
		if err != nil {
			break
		}
		return blueprints.CreateInput{UserGroups: []string{g.ID}}, "userGroups:" + g.ID, true
	}
	for u, err := range applebusiness.ListSeq[people.UserAttributes](ctx, c, "/v1/users", q) {
		if err != nil {
			break
		}
		return blueprints.CreateInput{Users: []string{u.ID}}, "users:" + u.ID, true
	}
	return blueprints.CreateInput{}, "", false
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
	fmt.Println("  Configurations: Create(CUSTOM_SETTING) → Update" + delNote(keep))
	fmt.Println("  Blueprints:     中身=上記Configuration＋割り当て先(userGroups/users)1件を一時付与（一瞬配信） → Create → Update" + relNote(appID) + delNote(keep))
	fmt.Println("  MdmServers:     Create(自己署名証明書・デバイス割り当てなし) → Get → Update(名前+defaultProductFamilies)" + delNote(keep))
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

// sampleMobileconfig は検証用の最小 .mobileconfig。
// 実機の検証は PayloadContent が空（<array/>）だと 400（'PayloadContent' failed on 'gt' tag）になるため、
// 無害な Web クリップ・ペイロードを1つだけ含める。Configuration 単体テストはどのデバイスにも適用せず即削除する。
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
  <key>PayloadContent</key>
  <array>
    <dict>
      <key>PayloadType</key><string>com.apple.webClip.managed</string>
      <key>PayloadVersion</key><integer>1</integer>
      <key>PayloadIdentifier</key><string>com.example.abmgo.` + name + `.webclip</string>
      <key>PayloadUUID</key><string>` + uuid() + `</string>
      <key>PayloadDisplayName</key><string>abm-go write-test</string>
      <key>URL</key><string>https://example.com</string>
      <key>Label</key><string>abm-go write-test</string>
    </dict>
  </array>
</dict>
</plist>`
}

// selfSignedCertBase64 は MDM サーバ登録用の自己署名 X.509 証明書を生成し、
// Base64（DER）で返す。鍵はこの場で捨てる（本物の MDM とペアリングする用途ではなく、
// CRUD の疎通確認のみが目的）。
func selfSignedCertBase64(cn string) string {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		fatalf("generate cert key: %v", err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		fatalf("generate cert serial: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		fatalf("create certificate: %v", err)
	}
	return base64.StdEncoding.EncodeToString(der)
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
