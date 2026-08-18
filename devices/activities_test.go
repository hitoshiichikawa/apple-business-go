package devices

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
	"github.com/hitoshiichikawa/apple-business-go/internal/testutil"
)

func TestAssignCreatesActivity(t *testing.T) {
	var body struct {
		Data struct {
			Type       string `json:"type"`
			Attributes struct {
				ActivityType string `json:"activityType"`
			} `json:"attributes"`
			Relationships struct {
				MdmServer struct {
					Data applebusiness.Data `json:"data"`
				} `json:"mdmServer"`
				Devices struct {
					Data []applebusiness.Data `json:"data"`
				} `json:"devices"`
			} `json:"relationships"`
		} `json:"data"`
	}
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/orgDeviceActivities" {
			http.Error(w, "unexpected "+r.Method+" "+r.URL.Path, http.StatusBadRequest)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		testutil.WriteJSON(t, w, http.StatusCreated, applebusiness.SingleResponse[ActivityAttributes]{
			Data: applebusiness.ResourceObject[ActivityAttributes]{Type: "orgDeviceActivities", ID: "ACT1", Attributes: ActivityAttributes{Status: StatusInProgress}},
		})
	})
	c := testutil.NewClient(t, h)
	act, err := New(c).Assign(context.Background(), "S1", []string{"D1", "D2"})
	if err != nil {
		t.Fatal(err)
	}
	if act.ID != "ACT1" || act.Attributes.Status != StatusInProgress {
		t.Fatalf("activity: %+v", act)
	}
	if body.Data.Type != "orgDeviceActivities" {
		t.Fatalf("type: %q", body.Data.Type)
	}
	if body.Data.Attributes.ActivityType != ActivityAssign {
		t.Fatalf("activityType: %q", body.Data.Attributes.ActivityType)
	}
	if body.Data.Relationships.MdmServer.Data.Type != "mdmServers" || body.Data.Relationships.MdmServer.Data.ID != "S1" {
		t.Fatalf("mdmServer rel: %+v", body.Data.Relationships.MdmServer.Data)
	}
	if len(body.Data.Relationships.Devices.Data) != 2 ||
		body.Data.Relationships.Devices.Data[0].Type != "orgDevices" ||
		body.Data.Relationships.Devices.Data[1].ID != "D2" {
		t.Fatalf("devices rel: %+v", body.Data.Relationships.Devices.Data)
	}
}

func TestUnassignActivityType(t *testing.T) {
	var activityType string
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b struct {
			Data struct {
				Attributes struct {
					ActivityType string `json:"activityType"`
				} `json:"attributes"`
			} `json:"data"`
		}
		_ = json.NewDecoder(r.Body).Decode(&b)
		activityType = b.Data.Attributes.ActivityType
		testutil.WriteJSON(t, w, http.StatusCreated, applebusiness.SingleResponse[ActivityAttributes]{
			Data: applebusiness.ResourceObject[ActivityAttributes]{Type: "orgDeviceActivities", ID: "ACT2", Attributes: ActivityAttributes{Status: StatusInProgress}},
		})
	})
	c := testutil.NewClient(t, h)
	if _, err := New(c).Unassign(context.Background(), "S1", []string{"D1"}); err != nil {
		t.Fatal(err)
	}
	if activityType != ActivityUnassign {
		t.Fatalf("activityType: %q want %q", activityType, ActivityUnassign)
	}
}

// migrationBody は移行系アクティビティのリクエスト形状検証用。生 JSON の
// キー有無（mdmServer / activityTypeMetadata の省略）まで見るため map で持つ。
type migrationBody struct {
	Data struct {
		Type       string `json:"type"`
		Attributes struct {
			ActivityType         string                     `json:"activityType"`
			ActivityTypeMetadata map[string]json.RawMessage `json:"activityTypeMetadata"`
		} `json:"attributes"`
		Relationships map[string]json.RawMessage `json:"relationships"`
	} `json:"data"`
}

func newActivityCaptureServer(t *testing.T, body *migrationBody) *applebusiness.Client {
	t.Helper()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/orgDeviceActivities" {
			http.Error(w, "unexpected "+r.Method+" "+r.URL.Path, http.StatusBadRequest)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(body)
		testutil.WriteJSON(t, w, http.StatusCreated, applebusiness.SingleResponse[ActivityAttributes]{
			Data: applebusiness.ResourceObject[ActivityAttributes]{Type: "orgDeviceActivities", ID: "ACT3", Attributes: ActivityAttributes{Status: StatusInProgress, SubStatus: SubStatusSubmitted}},
		})
	})
	return testutil.NewClient(t, h)
}

func TestAssignWithMdmMigrationDeadline(t *testing.T) {
	var body migrationBody
	c := newActivityCaptureServer(t, &body)
	deadline := time.Date(2026, 9, 15, 17, 0, 0, 0, time.UTC)
	act, err := New(c).AssignWithMdmMigrationDeadline(context.Background(), "S1", []string{"D1"}, deadline)
	if err != nil {
		t.Fatal(err)
	}
	if act.ID != "ACT3" {
		t.Fatalf("activity: %+v", act)
	}
	if body.Data.Attributes.ActivityType != ActivityAssignWithMdmMigrationDeadline {
		t.Fatalf("activityType: %q", body.Data.Attributes.ActivityType)
	}
	if got := string(body.Data.Attributes.ActivityTypeMetadata["mdmMigrationDeadlineDateTime"]); got != `"2026-09-15T17:00:00.000Z"` {
		t.Fatalf("mdmMigrationDeadlineDateTime: %s", got)
	}
	if _, ok := body.Data.Relationships["mdmServer"]; !ok {
		t.Fatalf("mdmServer relationship missing: %v", body.Data.Relationships)
	}
}

func TestUpdateMdmMigrationDeadlineOmitsMdmServer(t *testing.T) {
	var body migrationBody
	c := newActivityCaptureServer(t, &body)
	deadline := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	if _, err := New(c).UpdateMdmMigrationDeadline(context.Background(), []string{"D1", "D2"}, deadline); err != nil {
		t.Fatal(err)
	}
	if body.Data.Attributes.ActivityType != ActivityUpdateMdmMigrationDeadline {
		t.Fatalf("activityType: %q", body.Data.Attributes.ActivityType)
	}
	if got := string(body.Data.Attributes.ActivityTypeMetadata["mdmMigrationDeadlineDateTime"]); got != `"2026-10-01T00:00:00.000Z"` {
		t.Fatalf("mdmMigrationDeadlineDateTime: %s", got)
	}
	if _, ok := body.Data.Relationships["mdmServer"]; ok {
		t.Fatalf("mdmServer relationship should be omitted: %v", body.Data.Relationships)
	}
	if _, ok := body.Data.Relationships["devices"]; !ok {
		t.Fatalf("devices relationship missing: %v", body.Data.Relationships)
	}
}

func TestCancelMdmMigrationOmitsServerAndMetadata(t *testing.T) {
	var body migrationBody
	c := newActivityCaptureServer(t, &body)
	if _, err := New(c).CancelMdmMigration(context.Background(), []string{"D1"}); err != nil {
		t.Fatal(err)
	}
	if body.Data.Attributes.ActivityType != ActivityCancelMdmMigration {
		t.Fatalf("activityType: %q", body.Data.Attributes.ActivityType)
	}
	if body.Data.Attributes.ActivityTypeMetadata != nil {
		t.Fatalf("activityTypeMetadata should be omitted: %v", body.Data.Attributes.ActivityTypeMetadata)
	}
	if _, ok := body.Data.Relationships["mdmServer"]; ok {
		t.Fatalf("mdmServer relationship should be omitted: %v", body.Data.Relationships)
	}
}

func TestPollActivityUntilTerminal(t *testing.T) {
	var calls int
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		st, sub := StatusInProgress, ""
		if calls >= 2 {
			st, sub = StatusCompleted, SubStatusCompletedWithError
		}
		testutil.WriteJSON(t, w, http.StatusOK, applebusiness.SingleResponse[ActivityAttributes]{
			Data: applebusiness.ResourceObject[ActivityAttributes]{Type: "orgDeviceActivities", ID: "ACT1", Attributes: ActivityAttributes{Status: st, SubStatus: sub}},
		})
	})
	c := testutil.NewClient(t, h)
	final, err := New(c).PollActivity(context.Background(), "ACT1", time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if final.Attributes.Status != StatusCompleted || final.Attributes.SubStatus != SubStatusCompletedWithError {
		t.Fatalf("final: %+v", final.Attributes)
	}
	if calls < 2 {
		t.Fatalf("calls=%d", calls)
	}
}

func TestIsTerminalStatus(t *testing.T) {
	cases := map[string]bool{
		StatusInProgress: false,
		"PENDING":        false,
		StatusCompleted:  true,
		"FAILED":         true,
		"":               false,
	}
	for status, want := range cases {
		if got := isTerminalStatus(status); got != want {
			t.Errorf("isTerminalStatus(%q)=%v want %v", status, got, want)
		}
	}
}
