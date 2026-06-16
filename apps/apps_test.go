package apps

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
	"github.com/hitoshiichikawa/apple-business-go/internal/testutil"
)

func appsHandler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case p == "/v1/apps" && r.Method == http.MethodGet:
			testutil.WriteJSON(t, w, http.StatusOK, applebusiness.ListResponse[AppAttributes]{
				Data: []applebusiness.ResourceObject[AppAttributes]{
					{Type: "apps", ID: "A1", Attributes: AppAttributes{Name: "Pages", BundleID: "com.apple.Pages", SupportedOS: []string{OSIOS, OSIPadOS}}},
				},
			})
		case strings.HasPrefix(p, "/v1/apps/") && r.Method == http.MethodGet:
			id := strings.TrimPrefix(p, "/v1/apps/")
			testutil.WriteJSON(t, w, http.StatusOK, applebusiness.SingleResponse[AppAttributes]{
				Data: applebusiness.ResourceObject[AppAttributes]{Type: "apps", ID: id, Attributes: AppAttributes{Name: "Pages", BundleID: "com.apple.Pages"}},
			})
		case p == "/v1/packages" && r.Method == http.MethodGet:
			testutil.WriteJSON(t, w, http.StatusOK, applebusiness.ListResponse[PackageAttributes]{
				Data: []applebusiness.ResourceObject[PackageAttributes]{
					{Type: "packages", ID: "P1", Attributes: PackageAttributes{Name: "Bundle A", BundleIDs: []string{"com.x.a", "com.x.b"}}},
				},
			})
		case strings.HasPrefix(p, "/v1/packages/") && r.Method == http.MethodGet:
			id := strings.TrimPrefix(p, "/v1/packages/")
			testutil.WriteJSON(t, w, http.StatusOK, applebusiness.SingleResponse[PackageAttributes]{
				Data: applebusiness.ResourceObject[PackageAttributes]{Type: "packages", ID: id, Attributes: PackageAttributes{Name: "Bundle A"}},
			})
		default:
			http.Error(w, "not found: "+p, http.StatusNotFound)
		}
	}
}

func TestListApps(t *testing.T) {
	c := testutil.NewClient(t, appsHandler(t))
	got, err := New(c).ListApps(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "A1" || got[0].Attributes.BundleID != "com.apple.Pages" {
		t.Fatalf("unexpected: %+v", got)
	}
	if len(got[0].Attributes.SupportedOS) != 2 || got[0].Attributes.SupportedOS[0] != OSIOS {
		t.Fatalf("supportedOS: %+v", got[0].Attributes.SupportedOS)
	}
}

func TestGetApp(t *testing.T) {
	c := testutil.NewClient(t, appsHandler(t))
	got, err := New(c).GetApp(context.Background(), "A9")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "A9" || got.Attributes.Name != "Pages" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestListPackages(t *testing.T) {
	c := testutil.NewClient(t, appsHandler(t))
	got, err := New(c).ListPackages(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "P1" || len(got[0].Attributes.BundleIDs) != 2 {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestGetPackage(t *testing.T) {
	c := testutil.NewClient(t, appsHandler(t))
	got, err := New(c).GetPackage(context.Background(), "P9")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "P9" || got.Attributes.Name != "Bundle A" {
		t.Fatalf("unexpected: %+v", got)
	}
}
