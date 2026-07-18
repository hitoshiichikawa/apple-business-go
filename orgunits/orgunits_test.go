package orgunits

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
	"github.com/hitoshiichikawa/apple-business-go/internal/testutil"
)

func orgunitsHandler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case p == "/v1/organizationalUnits" && r.Method == http.MethodGet:
			if r.URL.Query().Get("cursor") == "p2" {
				testutil.WriteJSON(t, w, http.StatusOK, applebusiness.ListResponse[OrganizationalUnitAttributes]{
					Data: []applebusiness.ResourceObject[OrganizationalUnitAttributes]{
						{Type: "organizationalUnits", ID: "OU2", Attributes: OrganizationalUnitAttributes{Name: "Sales"}},
					},
				})
				return
			}
			testutil.WriteJSON(t, w, http.StatusOK, applebusiness.ListResponse[OrganizationalUnitAttributes]{
				Data: []applebusiness.ResourceObject[OrganizationalUnitAttributes]{
					{Type: "organizationalUnits", ID: "OU1", Attributes: OrganizationalUnitAttributes{Name: "HQ", Description: "Head office"}},
				},
				Links: applebusiness.Links{Next: "http://" + r.Host + "/v1/organizationalUnits?cursor=p2"},
			})
		case strings.HasSuffix(p, "/relationships/users"):
			testutil.WriteJSON(t, w, http.StatusOK, applebusiness.RelationshipResponse{
				Data: []applebusiness.Data{{Type: "users", ID: "U1"}, {Type: "users", ID: "U2"}},
			})
		case strings.HasPrefix(p, "/v1/organizationalUnits/") && r.Method == http.MethodGet:
			id := strings.TrimPrefix(p, "/v1/organizationalUnits/")
			testutil.WriteJSON(t, w, http.StatusOK, applebusiness.SingleResponse[OrganizationalUnitAttributes]{
				Data: applebusiness.ResourceObject[OrganizationalUnitAttributes]{Type: "organizationalUnits", ID: id, Attributes: OrganizationalUnitAttributes{Name: "HQ", Description: "Head office"}},
			})
		default:
			http.Error(w, "not found: "+p, http.StatusNotFound)
		}
	}
}

func TestList(t *testing.T) {
	c := testutil.NewClient(t, orgunitsHandler(t))
	got, err := New(c).List(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	// Two pages joined via links.next.
	if len(got) != 2 || got[0].ID != "OU1" || got[1].ID != "OU2" {
		t.Fatalf("unexpected: %+v", got)
	}
	if got[0].Type != "organizationalUnits" || got[0].Attributes.Name != "HQ" || got[0].Attributes.Description != "Head office" {
		t.Fatalf("attributes: %+v", got[0].Attributes)
	}
}

func TestGet(t *testing.T) {
	c := testutil.NewClient(t, orgunitsHandler(t))
	got, err := New(c).Get(context.Background(), "OU9")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "OU9" || got.Type != "organizationalUnits" || got.Attributes.Name != "HQ" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestMembers(t *testing.T) {
	c := testutil.NewClient(t, orgunitsHandler(t))
	ids, err := New(c).Members(context.Background(), "OU1")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0].Type != "users" || ids[1].ID != "U2" {
		t.Fatalf("unexpected: %+v", ids)
	}
}
