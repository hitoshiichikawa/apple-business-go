package people

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

func peopleHandler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case p == "/v1/users" && r.Method == http.MethodGet:
			writeJSON(t, w, http.StatusOK, applebusiness.ListResponse[UserAttributes]{
				Data: []applebusiness.ResourceObject[UserAttributes]{
					{Type: "users", ID: "U1", Attributes: UserAttributes{Email: "u1@example.com", FirstName: "Ichiro", Status: UserStatusActive}},
				},
			})
		case strings.HasPrefix(p, "/v1/users/") && r.Method == http.MethodGet:
			id := strings.TrimPrefix(p, "/v1/users/")
			writeJSON(t, w, http.StatusOK, applebusiness.SingleResponse[UserAttributes]{
				Data: applebusiness.ResourceObject[UserAttributes]{Type: "users", ID: id, Attributes: UserAttributes{Email: id + "@example.com", Status: UserStatusActive}},
			})
		case p == "/v1/userGroups" && r.Method == http.MethodGet:
			writeJSON(t, w, http.StatusOK, applebusiness.ListResponse[UserGroupAttributes]{
				Data: []applebusiness.ResourceObject[UserGroupAttributes]{
					{Type: "userGroups", ID: "G1", Attributes: UserGroupAttributes{Name: "All Staff", Type: UserGroupStandard, Status: "ACTIVE"}},
				},
			})
		case strings.HasSuffix(p, "/relationships/users"):
			writeJSON(t, w, http.StatusOK, applebusiness.RelationshipResponse{
				Data: []applebusiness.Data{{Type: "users", ID: "U1"}, {Type: "users", ID: "U2"}},
			})
		case strings.HasPrefix(p, "/v1/userGroups/") && r.Method == http.MethodGet:
			id := strings.TrimPrefix(p, "/v1/userGroups/")
			writeJSON(t, w, http.StatusOK, applebusiness.SingleResponse[UserGroupAttributes]{
				Data: applebusiness.ResourceObject[UserGroupAttributes]{Type: "userGroups", ID: id, Attributes: UserGroupAttributes{Name: "Smart Group", Type: UserGroupSmart}},
			})
		default:
			http.Error(w, "not found: "+p, http.StatusNotFound)
		}
	}
}

func TestListUsers(t *testing.T) {
	c := newClient(t, peopleHandler(t))
	got, err := New(c).ListUsers(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "U1" || got[0].Attributes.Email != "u1@example.com" || got[0].Attributes.Status != UserStatusActive {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestGetUser(t *testing.T) {
	c := newClient(t, peopleHandler(t))
	got, err := New(c).GetUser(context.Background(), "U9")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "U9" || got.Attributes.Email != "U9@example.com" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestListUserGroups(t *testing.T) {
	c := newClient(t, peopleHandler(t))
	got, err := New(c).ListUserGroups(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "G1" || got[0].Attributes.Type != UserGroupStandard {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestGetUserGroup(t *testing.T) {
	c := newClient(t, peopleHandler(t))
	got, err := New(c).GetUserGroup(context.Background(), "G9")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "G9" || got.Attributes.Type != UserGroupSmart {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestGroupMembers(t *testing.T) {
	c := newClient(t, peopleHandler(t))
	ids, err := New(c).GroupMembers(context.Background(), "G1")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0].Type != "users" || ids[1].ID != "U2" {
		t.Fatalf("unexpected: %+v", ids)
	}
}
