package blueprints

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
)

func testKeyPEM(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
}

func newClient(t *testing.T, h http.Handler) *applebusiness.Client {
	t.Helper()
	api := httptest.NewServer(h)
	t.Cleanup(api.Close)
	tok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"t","token_type":"Bearer","expires_in":3600}`))
	}))
	t.Cleanup(tok.Close)
	c, err := applebusiness.NewClient(applebusiness.Config{Credentials: applebusiness.Credentials{
		ClientID:   "BUSINESSAPI.test",
		TeamID:     "BUSINESSAPI.test",
		KeyID:      "kid",
		PrivateKey: testKeyPEM(t),
	}}, applebusiness.WithBaseURL(api.URL), applebusiness.WithTokenURL(tok.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatal(err)
	}
}

// writeCapture は書き込みボディの汎用キャプチャ。
type writeCapture struct {
	Data struct {
		Type          string                 `json:"type"`
		ID            string                 `json:"id"`
		Attributes    map[string]any         `json:"attributes"`
		Relationships map[string]relDataJSON `json:"relationships"`
	} `json:"data"`
}

type relDataJSON struct {
	Data []applebusiness.Data `json:"data"`
}

func TestBlueprintListGet(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/blueprints" && r.Method == http.MethodGet:
			writeJSON(t, w, http.StatusOK, applebusiness.ListResponse[Attributes]{
				Data: []applebusiness.ResourceObject[Attributes]{
					{Type: "blueprints", ID: "BP1", Attributes: Attributes{Name: "Default", Status: StatusActive}},
				},
			})
		case strings.HasSuffix(r.URL.Path, "/relationships/orgDevices"):
			writeJSON(t, w, http.StatusOK, applebusiness.RelationshipResponse{
				Data: []applebusiness.Data{{Type: "orgDevices", ID: "D1"}},
			})
		case strings.HasPrefix(r.URL.Path, "/v1/blueprints/") && r.Method == http.MethodGet:
			id := strings.TrimPrefix(r.URL.Path, "/v1/blueprints/")
			writeJSON(t, w, http.StatusOK, applebusiness.SingleResponse[Attributes]{
				Data: applebusiness.ResourceObject[Attributes]{Type: "blueprints", ID: id, Attributes: Attributes{Name: "Default", Status: StatusActive}},
			})
		default:
			http.Error(w, "not found: "+r.URL.Path, http.StatusNotFound)
		}
	})
	c := newClient(t, h)
	svc := New(c)

	list, err := svc.List(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != "BP1" || list[0].Attributes.Status != StatusActive {
		t.Fatalf("list: %+v", list)
	}

	got, err := svc.Get(context.Background(), "BP1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "BP1" || got.Attributes.Name != "Default" {
		t.Fatalf("get: %+v", got)
	}

	ids, err := svc.RelationshipIDs(context.Background(), "BP1", RelOrgDevices)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0].Type != "orgDevices" || ids[0].ID != "D1" {
		t.Fatalf("rel ids: %+v", ids)
	}
}

func TestBlueprintCreate(t *testing.T) {
	var got writeCapture
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/blueprints" {
			http.Error(w, "unexpected", http.StatusBadRequest)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		writeJSON(t, w, http.StatusCreated, applebusiness.SingleResponse[Attributes]{
			Data: applebusiness.ResourceObject[Attributes]{Type: "blueprints", ID: "BP9", Attributes: Attributes{Name: "New BP"}},
		})
	})
	c := newClient(t, h)
	bp, err := New(c).Create(context.Background(), CreateInput{Name: "New BP", OrgDevices: []string{"D1"}})
	if err != nil {
		t.Fatal(err)
	}
	if bp.ID != "BP9" {
		t.Fatalf("returned id: %s", bp.ID)
	}
	if got.Data.Type != "blueprints" {
		t.Fatalf("type: %q", got.Data.Type)
	}
	if got.Data.Attributes["name"] != "New BP" {
		t.Fatalf("attr name: %v", got.Data.Attributes)
	}
	rel, ok := got.Data.Relationships[RelOrgDevices]
	if !ok || len(rel.Data) != 1 || rel.Data[0].Type != "orgDevices" || rel.Data[0].ID != "D1" {
		t.Fatalf("relationships: %+v", got.Data.Relationships)
	}
}

func TestBlueprintUpdate(t *testing.T) {
	var got writeCapture
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/v1/blueprints/BP1" {
			http.Error(w, "unexpected "+r.Method+" "+r.URL.Path, http.StatusBadRequest)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		writeJSON(t, w, http.StatusOK, applebusiness.SingleResponse[Attributes]{
			Data: applebusiness.ResourceObject[Attributes]{Type: "blueprints", ID: "BP1", Attributes: Attributes{Name: "Renamed"}},
		})
	})
	c := newClient(t, h)
	name := "Renamed"
	if _, err := New(c).Update(context.Background(), "BP1", UpdateInput{Name: &name}); err != nil {
		t.Fatal(err)
	}
	if got.Data.ID != "BP1" || got.Data.Attributes["name"] != "Renamed" {
		t.Fatalf("update body: %+v", got.Data)
	}
}

func TestBlueprintDelete(t *testing.T) {
	var method, path string
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})
	c := newClient(t, h)
	if err := New(c).Delete(context.Background(), "BP1"); err != nil {
		t.Fatal(err)
	}
	if method != http.MethodDelete || path != "/v1/blueprints/BP1" {
		t.Fatalf("delete sent %s %s", method, path)
	}
}

func TestBlueprintRelationshipModify(t *testing.T) {
	type relCapture struct {
		Data []applebusiness.Data `json:"data"`
	}
	cases := []struct {
		name   string
		call   func(s *Service) error
		method string
	}{
		{"AddTo", func(s *Service) error { return s.AddTo(context.Background(), "BP1", RelApps, []string{"A1"}) }, http.MethodPost},
		{"RemoveFrom", func(s *Service) error { return s.RemoveFrom(context.Background(), "BP1", RelApps, []string{"A1"}) }, http.MethodDelete},
		{"Replace", func(s *Service) error { return s.Replace(context.Background(), "BP1", RelApps, []string{"A1"}) }, http.MethodPatch},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotMethod, gotPath string
			var body relCapture
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath = r.Method, r.URL.Path
				_ = json.NewDecoder(r.Body).Decode(&body)
				w.WriteHeader(http.StatusNoContent)
			})
			c := newClient(t, h)
			if err := tc.call(New(c)); err != nil {
				t.Fatal(err)
			}
			if gotMethod != tc.method {
				t.Fatalf("method = %s want %s", gotMethod, tc.method)
			}
			if gotPath != "/v1/blueprints/BP1/relationships/apps" {
				t.Fatalf("path = %s", gotPath)
			}
			if len(body.Data) != 1 || body.Data[0].Type != "apps" || body.Data[0].ID != "A1" {
				t.Fatalf("body data: %+v", body.Data)
			}
		})
	}
}
