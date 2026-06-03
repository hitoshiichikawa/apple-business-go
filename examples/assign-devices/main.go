// Command assign-devices は組織デバイスを MDM サーバへ割り当て/解除し、
// 生成されたアクティビティを完了までポーリングする。結果CSVの取得・保存も可能。
//
// ⚠️ 割り当て/解除は「書き込み」操作（実組織の MDM 割り当てを変更）。
// 既定はドライラン（無変更）。実行するには -yes を付ける。
//
// 認証情報は環境変数から（smoke-test / dump-all と同じ）:
//
//	AXM_CLIENT_ID, AXM_KEY_ID, AXM_PRIVATE_KEY_PATH（必須）
//	AXM_TEAM_ID, AXM_SCOPE, AXM_BASE_URL, AXM_TOKEN_URL（任意）
//
// 使い方:
//
//	# ドライラン（現状プレビュー・無変更）
//	go run ./examples/assign-devices -server <mdmServerId> -devices SERIAL1,SERIAL2
//	# 実行（割り当て）。完了後に結果CSVを保存
//	go run ./examples/assign-devices -server <mdmServerId> -devices SERIAL1 -yes -save-results result.csv
//	# 既存アクティビティの状態確認＋結果CSV取得（新規作成しない・書き込みなし）
//	go run ./examples/assign-devices -activity <activityId> -save-results result.csv
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
	"github.com/hitoshiichikawa/apple-business-go/devices"
)

func main() {
	server := flag.String("server", "", "対象 MDM サーバの ID")
	deviceCSV := flag.String("devices", "", "対象デバイスID(serial)をカンマ区切りで")
	unassign := flag.Bool("unassign", false, "割り当て解除する（既定は割り当て）")
	yes := flag.Bool("yes", false, "実際に実行する（未指定はドライラン＝無変更）")
	wait := flag.Bool("wait", true, "アクティビティ完了までポーリングする")
	activityID := flag.String("activity", "", "既存アクティビティID（指定時は新規作成せず状態確認/結果取得のみ）")
	saveResults := flag.String("save-results", "", "結果CSVの保存先パス（downloadUrl から取得）")
	timeout := flag.Duration("timeout", 5*time.Minute, "全体タイムアウト")
	flag.Parse()

	pem, err := os.ReadFile(os.Getenv("AXM_PRIVATE_KEY_PATH"))
	if err != nil {
		fatalf("read private key (AXM_PRIVATE_KEY_PATH): %v", err)
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
	}, applebusiness.WithUserAgent("apple-business-go/assign-devices"))
	if err != nil {
		fatalf("new client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	svc := devices.New(c)

	var final *devices.Activity

	switch {
	case *activityID != "":
		// 既存アクティビティ: 書き込みしない。再取得するので downloadUrl も新しい署名になる。
		if *wait {
			final, err = svc.PollActivity(ctx, *activityID, 3*time.Second)
		} else {
			final, err = svc.GetActivity(ctx, *activityID)
		}
		if err != nil {
			fatalf("activity: %v", err)
		}
		fmt.Printf("activity: id=%s status=%s subStatus=%s\n", final.ID, final.Attributes.Status, final.Attributes.SubStatus)

	default:
		// 新規作成パス（書き込み）。
		if *server == "" || *deviceCSV == "" {
			fatalf("usage: assign-devices -server <id> -devices <id[,id...]> [-unassign] [-yes]\n       または: assign-devices -activity <activityId> [-save-results <path>]")
		}
		var deviceIDs []string
		for _, p := range strings.Split(*deviceCSV, ",") {
			if s := strings.TrimSpace(p); s != "" {
				deviceIDs = append(deviceIDs, s)
			}
		}
		if len(deviceIDs) == 0 {
			fatalf("-devices に有効なIDがありません")
		}

		action := "ASSIGN（割り当て）"
		if *unassign {
			action = "UNASSIGN（解除）"
		}
		fmt.Printf("操作: %s\n対象サーバ: %s\n対象デバイス: %d 件\n", action, *server, len(deviceIDs))

		// プレビュー（読み取りのみ）
		for _, id := range deviceIDs {
			dev, derr := svc.Get(ctx, id)
			if derr != nil {
				fmt.Printf("  - %s : 取得失敗 (%v)\n", id, derr)
				continue
			}
			cur := "(未割り当て/不明)"
			if srv, serr := svc.AssignedServer(ctx, id); serr == nil && srv != nil && srv.ID != "" {
				cur = fmt.Sprintf("%s (%s)", srv.Attributes.ServerName, srv.ID)
			} else if applebusiness.IsNotFound(serr) {
				cur = "(未割り当て)"
			}
			fmt.Printf("  - %s  serial=%s model=%s status=%s  現在の割り当て先=%s\n",
				id, dev.Attributes.SerialNumber, dev.Attributes.DeviceModel, dev.Attributes.Status, cur)
		}

		if !*yes {
			fmt.Println("\nドライラン（変更していません）。実行するには -yes を付けて再実行してください。")
			return
		}

		var act *devices.Activity
		if *unassign {
			act, err = svc.Unassign(ctx, *server, deviceIDs)
		} else {
			act, err = svc.Assign(ctx, *server, deviceIDs)
		}
		if err != nil {
			fatalf("create activity: %v", err)
		}
		fmt.Printf("\n✓ activity created: id=%s status=%s\n", act.ID, act.Attributes.Status)
		final = act

		if *wait {
			final, err = svc.PollActivity(ctx, act.ID, 3*time.Second)
			if err != nil {
				fatalf("poll activity: %v", err)
			}
			fmt.Printf("✓ final: status=%s subStatus=%s\n", final.Attributes.Status, final.Attributes.SubStatus)
		} else {
			fmt.Printf("(-wait=false のため完了待ちしません。-activity %s で後から確認可能)\n", act.ID)
		}
	}

	// 結果CSVの取得・保存
	url := ""
	if final != nil {
		url = final.Attributes.DownloadURL
	}
	switch {
	case *saveResults != "" && url != "":
		if err := downloadFile(ctx, url, *saveResults); err != nil {
			fatalf("download results: %v", err)
		}
		fmt.Printf("✓ 結果CSVを保存しました: %s\n", *saveResults)
	case *saveResults != "":
		fmt.Println("結果URL(downloadUrl)がありません（まだ生成されていない可能性）。")
	case url != "":
		fmt.Printf("  results(URL): %s\n", url)
		fmt.Println("  ↑ curl で取得する場合は & を含むため必ずシングルクォートで囲む。あるいは -save-results <path> を使う。")
	}
}

// downloadFile は署名付きURL（認証不要）から内容を取得して path に保存する。
// 注意: SDK クライアントは使わない（Bearer を Apple の blobstore に送らないため）。
func downloadFile(ctx context.Context, rawurl, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawurl, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return err
	}
	return nil
}

func fatalf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", a...)
	os.Exit(1)
}
