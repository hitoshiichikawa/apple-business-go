package devices

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
	"github.com/hitoshiichikawa/apple-business-go/internal/testutil"
)

// mdmServerWriteCapture は mdmServers 書き込みボディの汎用キャプチャ。
type mdmServerWriteCapture struct {
	Data struct {
		Type       string         `json:"type"`
		ID         string         `json:"id"`
		Attributes map[string]any `json:"attributes"`
	} `json:"data"`
}

func TestGetMdmServer(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/mdmServers/S1" {
			http.Error(w, "unexpected "+r.Method+" "+r.URL.Path, http.StatusBadRequest)
			return
		}
		testutil.WriteJSON(t, w, http.StatusOK, applebusiness.SingleResponse[MdmServerAttributes]{
			Data: applebusiness.ResourceObject[MdmServerAttributes]{Type: "mdmServers", ID: "S1", Attributes: MdmServerAttributes{
				ServerName: "Server-1", ServerType: "MDM",
				Status: MdmServerStatusActive, DeviceCount: 12, EnableMdmDisownFlag: true,
				DefaultProductFamilies: []string{MdmProductFamilyIPhone, MdmProductFamilyIPad},
				LastConnectedDateTime:  "2026-07-01T00:00:00Z", LastConnectedIP: "203.0.113.10",
			}},
		})
	})
	c := testutil.NewClient(t, h)
	got, err := New(c).GetMdmServer(context.Background(), "S1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "S1" || got.Attributes.ServerName != "Server-1" {
		t.Fatalf("unexpected: %+v", got)
	}
	a := got.Attributes
	if a.Status != MdmServerStatusActive || a.DeviceCount != 12 || !a.EnableMdmDisownFlag ||
		len(a.DefaultProductFamilies) != 2 || a.DefaultProductFamilies[0] != MdmProductFamilyIPhone ||
		a.LastConnectedIP != "203.0.113.10" {
		t.Fatalf("attributes: %+v", a)
	}
}

func TestCreateMdmServer(t *testing.T) {
	var got mdmServerWriteCapture
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/mdmServers" {
			http.Error(w, "unexpected "+r.Method+" "+r.URL.Path, http.StatusBadRequest)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		testutil.WriteJSON(t, w, http.StatusCreated, applebusiness.SingleResponse[MdmServerAttributes]{
			Data: applebusiness.ResourceObject[MdmServerAttributes]{Type: "mdmServers", ID: "S9", Attributes: MdmServerAttributes{
				ServerName: "New MDM", ServerType: "MDM", Status: MdmServerStatusActive,
			}},
		})
	})
	c := testutil.NewClient(t, h)
	flag := true
	srv, err := New(c).CreateMdmServer(context.Background(), CreateMdmServerInput{
		ServerName:          "New MDM",
		ServerCertificate:   MdmServerCertificate{Name: "mdm.cer", Data: "QkFTRTY0"},
		EnableMdmDisownFlag: &flag,
	})
	if err != nil {
		t.Fatal(err)
	}
	if srv.ID != "S9" || srv.Attributes.Status != MdmServerStatusActive {
		t.Fatalf("returned: %+v", srv)
	}
	if got.Data.Type != "mdmServers" || got.Data.ID != "" {
		t.Fatalf("data: %+v", got.Data)
	}
	if got.Data.Attributes["serverName"] != "New MDM" {
		t.Fatalf("attr serverName: %v", got.Data.Attributes)
	}
	cert, ok := got.Data.Attributes["serverCertificate"].(map[string]any)
	if !ok || cert["name"] != "mdm.cer" || cert["data"] != "QkFTRTY0" {
		t.Fatalf("attr serverCertificate: %v", got.Data.Attributes)
	}
	if got.Data.Attributes["enableMdmDisownFlag"] != true {
		t.Fatalf("attr enableMdmDisownFlag: %v", got.Data.Attributes)
	}
}

func TestCreateMdmServerOmitsOptionalFlag(t *testing.T) {
	var got mdmServerWriteCapture
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		testutil.WriteJSON(t, w, http.StatusCreated, applebusiness.SingleResponse[MdmServerAttributes]{
			Data: applebusiness.ResourceObject[MdmServerAttributes]{Type: "mdmServers", ID: "S9"},
		})
	})
	c := testutil.NewClient(t, h)
	_, err := New(c).CreateMdmServer(context.Background(), CreateMdmServerInput{
		ServerName:        "New MDM",
		ServerCertificate: MdmServerCertificate{Name: "mdm.cer", Data: "QkFTRTY0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, present := got.Data.Attributes["enableMdmDisownFlag"]; present {
		t.Fatalf("enableMdmDisownFlag should be omitted: %v", got.Data.Attributes)
	}
}

func TestUpdateMdmServer(t *testing.T) {
	var got mdmServerWriteCapture
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/v1/mdmServers/S1" {
			http.Error(w, "unexpected "+r.Method+" "+r.URL.Path, http.StatusBadRequest)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		testutil.WriteJSON(t, w, http.StatusOK, applebusiness.SingleResponse[MdmServerAttributes]{
			Data: applebusiness.ResourceObject[MdmServerAttributes]{Type: "mdmServers", ID: "S1", Attributes: MdmServerAttributes{
				ServerName: "Renamed", DefaultProductFamilies: []string{MdmProductFamilyMac},
			}},
		})
	})
	c := testutil.NewClient(t, h)
	name := "Renamed"
	srv, err := New(c).UpdateMdmServer(context.Background(), "S1", UpdateMdmServerInput{
		ServerName:             &name,
		DefaultProductFamilies: []string{MdmProductFamilyMac},
	})
	if err != nil {
		t.Fatal(err)
	}
	if srv.ID != "S1" || srv.Attributes.ServerName != "Renamed" {
		t.Fatalf("returned: %+v", srv)
	}
	if got.Data.Type != "mdmServers" || got.Data.ID != "S1" {
		t.Fatalf("data: %+v", got.Data)
	}
	if got.Data.Attributes["serverName"] != "Renamed" {
		t.Fatalf("attr serverName: %v", got.Data.Attributes)
	}
	fams, ok := got.Data.Attributes["defaultProductFamilies"].([]any)
	if !ok || len(fams) != 1 || fams[0] != MdmProductFamilyMac {
		t.Fatalf("attr defaultProductFamilies: %v", got.Data.Attributes)
	}
	// 未指定の属性は送信されない（部分更新）。
	for _, k := range []string{"serverCertificate", "enableMdmDisownFlag"} {
		if _, present := got.Data.Attributes[k]; present {
			t.Fatalf("%s should be omitted: %v", k, got.Data.Attributes)
		}
	}
}

func TestDeleteMdmServer(t *testing.T) {
	var method, path string
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})
	c := testutil.NewClient(t, h)
	if err := New(c).DeleteMdmServer(context.Background(), "S1"); err != nil {
		t.Fatal(err)
	}
	if method != http.MethodDelete || path != "/v1/mdmServers/S1" {
		t.Fatalf("request: %s %s", method, path)
	}
}
