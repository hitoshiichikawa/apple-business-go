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
			ActivityType         string                `json:"activityType"`
			ActivityTypeMetadata *ActivityTypeMetadata `json:"activityTypeMetadata,omitempty"`
		} `json:"attributes"`
		Relationships struct {
			MdmServer *struct {
				Data applebusiness.Data `json:"data"`
			} `json:"mdmServer,omitempty"`
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
	return s.createActivity(ctx, ActivityAssign, serverID, deviceIDs, nil)
}

// Unassign removes devices from an MDM server.
func (s *Service) Unassign(ctx context.Context, serverID string, deviceIDs []string) (*Activity, error) {
	return s.createActivity(ctx, ActivityUnassign, serverID, deviceIDs, nil)
}

// AssignWithMdmMigrationDeadline assigns devices to an MDM server and schedules
// a device management service migration to it, to complete by deadline
// (API 2.3+). The deadline must be within 90 days; outside the allowed range
// the API returns 409. Devices must currently be enrolled in another device
// management service and be migration-capable (DeviceAttributes.IsMdmMigrationCapable).
func (s *Service) AssignWithMdmMigrationDeadline(ctx context.Context, serverID string, deviceIDs []string, deadline time.Time) (*Activity, error) {
	meta := &ActivityTypeMetadata{MdmMigrationDeadlineDateTime: formatDeadline(deadline)}
	return s.createActivity(ctx, ActivityAssignWithMdmMigrationDeadline, serverID, deviceIDs, meta)
}

// UpdateMdmMigrationDeadline updates the deadline of an in-progress device
// management service migration for the given devices (API 2.3+). No MDM server
// is specified; the devices' pending migration is updated in place.
func (s *Service) UpdateMdmMigrationDeadline(ctx context.Context, deviceIDs []string, deadline time.Time) (*Activity, error) {
	meta := &ActivityTypeMetadata{MdmMigrationDeadlineDateTime: formatDeadline(deadline)}
	return s.createActivity(ctx, ActivityUpdateMdmMigrationDeadline, "", deviceIDs, meta)
}

// CancelMdmMigration cancels an in-progress device management service
// migration for the given devices (API 2.3+).
func (s *Service) CancelMdmMigration(ctx context.Context, deviceIDs []string) (*Activity, error) {
	return s.createActivity(ctx, ActivityCancelMdmMigration, "", deviceIDs, nil)
}

// ReleaseDevices releases the given devices from the organization (API 2.4+).
// Destructive: released devices are no longer registered to the organization,
// their device enrollment assignments are removed, they are unenrolled from
// the built-in device management service (Apple MDM), and they are removed
// from Blueprints. This cannot be undone via the API.
func (s *Service) ReleaseDevices(ctx context.Context, deviceIDs []string) (*Activity, error) {
	return s.createActivity(ctx, ActivityReleaseDevices, "", deviceIDs, nil)
}

// formatDeadline renders an ISO 8601 timestamp with millisecond precision,
// matching Apple's documented examples (e.g. 2026-03-15T17:00:00.000Z).
func formatDeadline(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z07:00")
}

// createActivity posts an orgDeviceActivity. serverID may be empty for
// activity types that don't take an mdmServer relationship
// (UPDATE_MDM_MIGRATION_DEADLINE / CANCEL_MDM_MIGRATION / RELEASE_DEVICES).
func (s *Service) createActivity(ctx context.Context, activityType, serverID string, deviceIDs []string, meta *ActivityTypeMetadata) (*Activity, error) {
	var body activityCreate
	body.Data.Type = "orgDeviceActivities"
	body.Data.Attributes.ActivityType = activityType
	body.Data.Attributes.ActivityTypeMetadata = meta
	if serverID != "" {
		body.Data.Relationships.MdmServer = &struct {
			Data applebusiness.Data `json:"data"`
		}{Data: applebusiness.Data{Type: "mdmServers", ID: serverID}}
	}
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

// Activity statuses (official values per the OrgDeviceActivity.Attributes docs).
//
//	status:    IN_PROGRESS (processing) -> COMPLETED / STOPPED / FAILED
//	           (COMPLETED even on partial failure; see subStatus)
//	subStatus: COMPLETED_WITH_ERROR means some items failed (see the CSV at downloadUrl).
const (
	StatusInProgress = "IN_PROGRESS"
	StatusCompleted  = "COMPLETED"
	StatusStopped    = "STOPPED"
	StatusFailed     = "FAILED"

	SubStatusSubmitted                     = "SUBMITTED"
	SubStatusPreProcessing                 = "PRE_PROCESSING"
	SubStatusPending                       = "PENDING"
	SubStatusProcessing                    = "PROCESSING"
	SubStatusPostProcessing                = "POST_PROCESSING"
	SubStatusStopping                      = "STOPPING"
	SubStatusCompletedWithSuccess          = "COMPLETED_WITH_SUCCESS"
	SubStatusCompletedWithError            = "COMPLETED_WITH_ERROR" // observed: some failed
	SubStatusCompletedWithFailure          = "COMPLETED_WITH_FAILURE"
	SubStatusCompletedPostProcessingFailed = "COMPLETED_POST_PROCESSING_FAILED"
)

// SubStatusCompleted was a presumed full-success value that isn't in the
// official enum.
//
// Deprecated: use SubStatusCompletedWithSuccess.
const SubStatusCompleted = "COMPLETED"

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
