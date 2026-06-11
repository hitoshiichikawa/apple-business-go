package people

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
	"github.com/hitoshiichikawa/apple-business-go/internal/testutil"
)

func peopleHandler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case p == "/v1/users" && r.Method == http.MethodGet:
			testutil.WriteJSON(t, w, http.StatusOK, applebusiness.ListResponse[UserAttributes]{
				Data: []applebusiness.ResourceObject[UserAttributes]{
					{Type: "users", ID: "U1", Attributes: UserAttributes{Email: "u1@example.com", FirstName: "Ichiro", Status: UserStatusActive}},
				},
			})
		case strings.HasPrefix(p, "/v1/users/") && r.Method == http.MethodGet:
			id := strings.TrimPrefix(p, "/v1/users/")
			testutil.WriteJSON(t, w, http.StatusOK, applebusiness.SingleResponse[UserAttributes]{
				Data: applebusiness.ResourceObject[UserAttributes]{Type: "users", ID: id, Attributes: UserAttributes{Email: id + "@example.com", Status: UserStatusActive}},
			})
		case p == "/v1/userGroups" && r.Method == http.MethodGet:
			testutil.WriteJSON(t, w, http.StatusOK, applebusiness.ListResponse[UserGroupAttributes]{
				Data: []applebusiness.ResourceObject[UserGroupAttributes]{
					{Type: "userGroups", ID: "G1", Attributes: UserGroupAttributes{Name: "All Staff", Type: UserGroupStandard, Status: "ACTIVE"}},
				},
			})
		case strings.HasSuffix(p, "/relationships/users"):
			testutil.WriteJSON(t, w, http.StatusOK, applebusiness.RelationshipResponse{
				Data: []applebusiness.Data{{Type: "users", ID: "U1"}, {Type: "users", ID: "U2"}},
			})
		case strings.HasPrefix(p, "/v1/userGroups/") && r.Method == http.MethodGet:
			id := strings.TrimPrefix(p, "/v1/userGroups/")
			testutil.WriteJSON(t, w, http.StatusOK, applebusiness.SingleResponse[UserGroupAttributes]{
				Data: applebusiness.ResourceObject[UserGroupAttributes]{Type: "userGroups", ID: id, Attributes: UserGroupAttributes{Name: "Smart Group", Type: UserGroupSmart}},
			})
		default:
			http.Error(w, "not found: "+p, http.StatusNotFound)
		}
	}
}

func TestListUsers(t *testing.T) {
	c := testutil.NewClient(t, peopleHandler(t))
	got, err := New(c).ListUsers(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "U1" || got[0].Attributes.Email != "u1@example.com" || got[0].Attributes.Status != UserStatusActive {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestGetUser(t *testing.T) {
	c := testutil.NewClient(t, peopleHandler(t))
	got, err := New(c).GetUser(context.Background(), "U9")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "U9" || got.Attributes.Email != "U9@example.com" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestListUserGroups(t *testing.T) {
	c := testutil.NewClient(t, peopleHandler(t))
	got, err := New(c).ListUserGroups(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "G1" || got[0].Attributes.Type != UserGroupStandard {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestGetUserGroup(t *testing.T) {
	c := testutil.NewClient(t, peopleHandler(t))
	got, err := New(c).GetUserGroup(context.Background(), "G9")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "G9" || got.Attributes.Type != UserGroupSmart {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestGroupMembers(t *testing.T) {
	c := testutil.NewClient(t, peopleHandler(t))
	ids, err := New(c).GroupMembers(context.Background(), "G1")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0].Type != "users" || ids[1].ID != "U2" {
		t.Fatalf("unexpected: %+v", ids)
	}
}
