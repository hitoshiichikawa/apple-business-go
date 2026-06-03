package configurations

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

func TestConfigurationListGet(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/configurations" && r.Method == http.MethodGet:
			writeJSON(t, w, http.StatusOK, applebusiness.ListResponse[Attributes]{
				Data: []applebusiness.ResourceObject[Attributes]{
					{Type: "configurations", ID: "C1", Attributes: Attributes{Type: TypeCustomSetting, Name: "WiFi", CustomSettingsValues: &CustomSettingsValues{Filename: "wifi.mobileconfig"}}},
				},
			})
		case strings.HasPrefix(r.URL.Path, "/v1/configurations/") && r.Method == http.MethodGet:
			id := strings.TrimPrefix(r.URL.Path, "/v1/configurations/")
			writeJSON(t, w, http.StatusOK, applebusiness.SingleResponse[Attributes]{
				Data: applebusiness.ResourceObject[Attributes]{Type: "configurations", ID: id, Attributes: Attributes{Type: TypeCustomSetting, Name: "WiFi"}},
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
	if len(list) != 1 || list[0].Attributes.Type != TypeCustomSetting {
		t.Fatalf("list: %+v", list)
	}
	if list[0].Attributes.CustomSettingsValues == nil || list[0].Attributes.CustomSettingsValues.Filename != "wifi.mobileconfig" {
		t.Fatalf("csv: %+v", list[0].Attributes.CustomSettingsValues)
	}

	got, err := svc.Get(context.Background(), "C1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "C1" || got.Attributes.Name != "WiFi" {
		t.Fatalf("get: %+v", got)
	}
}

func TestConfigurationCreate(t *testing.T) {
	var got struct {
		Data struct {
			Type       string `json:"type"`
			ID         string `json:"id"`
			Attributes struct {
				Type                 string `json:"type"`
				Name                 string `json:"name"`
				CustomSettingsValues struct {
					ConfigurationProfile string `json:"configurationProfile"`
					Filename             string `json:"filename"`
				} `json:"customSettingsValues"`
			} `json:"attributes"`
		} `json:"data"`
	}
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/configurations" {
			http.Error(w, "unexpected", http.StatusBadRequest)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		writeJSON(t, w, http.StatusCreated, applebusiness.SingleResponse[Attributes]{
			Data: applebusiness.ResourceObject[Attributes]{Type: "configurations", ID: "C9", Attributes: Attributes{Type: TypeCustomSetting, Name: "New"}},
		})
	})
	c := newClient(t, h)
	cfg, err := New(c).Create(context.Background(), CreateInput{
		Name:                 "New",
		ConfigurationProfile: "<plist/>",
		Filename:             "new.mobileconfig",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ID != "C9" {
		t.Fatalf("returned id: %s", cfg.ID)
	}
	if got.Data.Type != "configurations" {
		t.Fatalf("data.type: %q", got.Data.Type)
	}
	if got.Data.Attributes.Type != TypeCustomSetting || got.Data.Attributes.Name != "New" {
		t.Fatalf("attrs: %+v", got.Data.Attributes)
	}
	if got.Data.Attributes.CustomSettingsValues.ConfigurationProfile != "<plist/>" ||
		got.Data.Attributes.CustomSettingsValues.Filename != "new.mobileconfig" {
		t.Fatalf("csv: %+v", got.Data.Attributes.CustomSettingsValues)
	}
}

func TestConfigurationUpdateDelete(t *testing.T) {
	// Update
	var upMethod, upPath, upName, upID string
	hUp := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upMethod, upPath = r.Method, r.URL.Path
		var b struct {
			Data struct {
				ID         string `json:"id"`
				Attributes struct {
					Name string `json:"name"`
				} `json:"attributes"`
			} `json:"data"`
		}
		_ = json.NewDecoder(r.Body).Decode(&b)
		upName, upID = b.Data.Attributes.Name, b.Data.ID
		writeJSON(t, w, http.StatusOK, applebusiness.SingleResponse[Attributes]{
			Data: applebusiness.ResourceObject[Attributes]{Type: "configurations", ID: "C1", Attributes: Attributes{Name: "N2"}},
		})
	})
	c := newClient(t, hUp)
	newName := "N2"
	if _, err := New(c).Update(context.Background(), "C1", UpdateInput{Name: &newName}); err != nil {
		t.Fatal(err)
	}
	if upMethod != http.MethodPatch || upPath != "/v1/configurations/C1" {
		t.Fatalf("update sent %s %s", upMethod, upPath)
	}
	if upID != "C1" || upName != "N2" {
		t.Fatalf("update body id=%q name=%q", upID, upName)
	}

	// Delete
	var delMethod, delPath string
	hDel := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		delMethod, delPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})
	c2 := newClient(t, hDel)
	if err := New(c2).Delete(context.Background(), "C1"); err != nil {
		t.Fatal(err)
	}
	if delMethod != http.MethodDelete || delPath != "/v1/configurations/C1" {
		t.Fatalf("delete sent %s %s", delMethod, delPath)
	}
}
