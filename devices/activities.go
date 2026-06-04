package devices

import (
	"context"
	"net/url"
	"time"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
)

// 書き込み（割り当て/解除）。アプリ層では admin 限定 + 監査を推奨。

type activityCreate struct {
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

// Assign assigns devices to an MDM server. Poll the returned Activity for completion.
// Note: specifying devices already assigned to the same server yields
// subStatus=COMPLETED_WITH_ERROR for those (their state is unchanged).
func (s *Service) Assign(ctx context.Context, serverID string, deviceIDs []string) (*Activity, error) {
	return s.createActivity(ctx, ActivityAssign, serverID, deviceIDs)
}

// Unassign removes devices from an MDM server.
func (s *Service) Unassign(ctx context.Context, serverID string, deviceIDs []string) (*Activity, error) {
	return s.createActivity(ctx, ActivityUnassign, serverID, deviceIDs)
}

func (s *Service) createActivity(ctx context.Context, activityType, serverID string, deviceIDs []string) (*Activity, error) {
	var body activityCreate
	body.Data.Type = "orgDeviceActivities"
	body.Data.Attributes.ActivityType = activityType
	body.Data.Relationships.MdmServer.Data = applebusiness.Data{Type: "mdmServers", ID: serverID}
	for _, id := range deviceIDs {
		body.Data.Relationships.Devices.Data = append(
			body.Data.Relationships.Devices.Data, applebusiness.Data{Type: "orgDevices", ID: id})
	}
	return applebusiness.Create[ActivityAttributes](ctx, s.c, "/v1/orgDeviceActivities", body)
}

// GetActivity returns the current status of an activity.
func (s *Service) GetActivity(ctx context.Context, activityID string) (*Activity, error) {
	return applebusiness.Get[ActivityAttributes](ctx, s.c, "/v1/orgDeviceActivities/"+url.PathEscape(activityID))
}

// Activity statuses (values observed against the live API).
//
//	status:    IN_PROGRESS (processing) -> COMPLETED (done; COMPLETED even on partial failure)
//	subStatus: COMPLETED_WITH_ERROR (some failed; see the CSV at downloadUrl). On full success it is presumably COMPLETED (not observed).
const (
	StatusInProgress = "IN_PROGRESS"
	StatusCompleted  = "COMPLETED"

	SubStatusCompleted          = "COMPLETED"            // presumed: all succeeded (not observed)
	SubStatusCompletedWithError = "COMPLETED_WITH_ERROR" // observed: some failed
)

// inProgressStatuses は「処理中（非終了）」とみなす status。
// status 文字列の網羅は不明なため、これ以外を終了とみなす（未知の終了 status で無限ループしないように）。
var inProgressStatuses = map[string]bool{
	StatusInProgress: true,
	"PENDING":        true,
	"QUEUED":         true,
	"STARTED":        true,
	"RUNNING":        true,
}

// isTerminalStatus は status が終了状態かを返す（空でなく、処理中集合に無ければ終了）。
func isTerminalStatus(status string) bool {
	return status != "" && !inProgressStatuses[status]
}

// PollActivity polls until a terminal status or ctx cancellation.
// Even when completed, a subStatus of COMPLETED_WITH_ERROR means some items failed (check the CSV at downloadUrl).
func (s *Service) PollActivity(ctx context.Context, activityID string, interval time.Duration) (*Activity, error) {
	if interval <= 0 {
		interval = 3 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		a, err := s.GetActivity(ctx, activityID)
		if err != nil {
			return nil, err
		}
		if isTerminalStatus(a.Attributes.Status) {
			return a, nil
		}
		select {
		case <-ctx.Done():
			return a, ctx.Err()
		case <-ticker.C:
		}
	}
}
