package devices

import "github.com/hitoshiichikawa/apple-business-go/applebusiness"

// Type aliases (keep return types concise).
type (
	Device    = applebusiness.ResourceObject[DeviceAttributes]
	MdmServer = applebusiness.ResourceObject[MdmServerAttributes]
	Coverage  = applebusiness.ResourceObject[AppleCareCoverageAttributes]
	Activity  = applebusiness.ResourceObject[ActivityAttributes]
)

// Values of OrgDeviceActivityType. The MDM-migration variants require API 2.3+
// (device management service migration, 2026-08-12); RELEASE_DEVICES requires
// API 2.4+ (2026-08-26).
const (
	ActivityAssign                         = "ASSIGN_DEVICES"
	ActivityUnassign                       = "UNASSIGN_DEVICES"
	ActivityAssignWithMdmMigrationDeadline = "ASSIGN_DEVICES_WITH_MDM_MIGRATION_DEADLINE"
	ActivityUpdateMdmMigrationDeadline     = "UPDATE_MDM_MIGRATION_DEADLINE"
	ActivityCancelMdmMigration             = "CANCEL_MDM_MIGRATION"
	ActivityReleaseDevices                 = "RELEASE_DEVICES"
)

// DeviceAttributes : /v1/orgDevices
type DeviceAttributes struct {
	SerialNumber            string   `json:"serialNumber"`
	DeviceModel             string   `json:"deviceModel"`
	ProductFamily           string   `json:"productFamily"` // iPhone/iPad/Mac/AppleTV/Watch/Vision
	ProductType             string   `json:"productType"`
	Color                   string   `json:"color"`
	DeviceCapacity          string   `json:"deviceCapacity"`
	PartNumber              string   `json:"partNumber,omitempty"`
	OrderNumber             string   `json:"orderNumber,omitempty"`
	OrderDateTime           string   `json:"orderDateTime,omitempty"`
	AddedToOrgDateTime      string   `json:"addedToOrgDateTime"`
	ReleasedFromOrgDateTime string   `json:"releasedFromOrgDateTime,omitempty"`
	Status                  string   `json:"status"` // ASSIGNED | UNASSIGNED
	WifiMacAddress          string   `json:"wifiMacAddress,omitempty"`
	BluetoothMacAddress     string   `json:"bluetoothMacAddress,omitempty"`
	EthernetMacAddress      []string `json:"ethernetMacAddress,omitempty"`
	IMEI                    []string `json:"imei,omitempty"`
	MEID                    []string `json:"meid,omitempty"`
	EID                     string   `json:"eid,omitempty"`
	PurchaseSourceID        string   `json:"purchaseSourceId"`
	PurchaseSourceType      string   `json:"purchaseSourceType"`
	UpdatedDateTime         string   `json:"updatedDateTime"`
	ReleaserEntityType      string   `json:"releaserEntityType,omitempty"`
	ReleaserID              string   `json:"releaserId,omitempty"`

	// Device management service migration (API 2.3+). The status/deadline
	// fields are set only when a migration has been requested.
	IsMdmMigrationCapable        bool   `json:"isMdmMigrationCapable,omitempty"`
	MdmMigrationStatus           string `json:"mdmMigrationStatus,omitempty"` // MdmMigrationStatus: REQUESTED | STARTED | SUCCESS | FAILED
	MdmMigrationDeadlineDateTime string `json:"mdmMigrationDeadlineDateTime,omitempty"`
}

// Values of MdmMigrationStatus (DeviceAttributes.MdmMigrationStatus).
const (
	MdmMigrationStatusRequested = "REQUESTED"
	MdmMigrationStatusStarted   = "STARTED"
	MdmMigrationStatusSuccess   = "SUCCESS"
	MdmMigrationStatusFailed    = "FAILED"
)

// MdmServerAttributes : /v1/mdmServers and /v1/orgDevices/{id}/assignedServer
type MdmServerAttributes struct {
	ServerName             string   `json:"serverName"`
	ServerType             string   `json:"serverType"`                       // MDM | APPLE_CONFIGURATOR | APPLE_MDM
	Status                 string   `json:"status,omitempty"`                 // MdmServerStatus: ACTIVE | INACTIVE | DELETED
	DeviceCount            int64    `json:"deviceCount,omitempty"`            // read only
	EnableMdmDisownFlag    bool     `json:"enableMdmDisownFlag,omitempty"`    // service may disown its enrolled devices
	DefaultProductFamilies []string `json:"defaultProductFamilies,omitempty"` // MdmServerProductFamily values
	LastConnectedDateTime  string   `json:"lastConnectedDateTime,omitempty"`  // read only
	LastConnectedIP        string   `json:"lastConnectedIp,omitempty"`        // read only
	CreatedDateTime        string   `json:"createdDateTime"`
	UpdatedDateTime        string   `json:"updatedDateTime"`
}

// MdmServerCertificate is the X.509 certificate to associate with a device
// management service (Create/Update MdmServer). Data is the Base64-encoded
// certificate.
type MdmServerCertificate struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

// Values of MdmServerStatus.
const (
	MdmServerStatusActive   = "ACTIVE"
	MdmServerStatusInactive = "INACTIVE"
	MdmServerStatusDeleted  = "DELETED"
)

// Values of MdmServerProductFamily (MdmServerAttributes.DefaultProductFamilies).
const (
	MdmProductFamilyAppleTV = "APPLE_TV"
	MdmProductFamilyIPad    = "IPAD"
	MdmProductFamilyIPhone  = "IPHONE"
	MdmProductFamilyIPod    = "IPOD"
	MdmProductFamilyMac     = "MAC"
	MdmProductFamilyVision  = "VISION"
	MdmProductFamilyWatch   = "WATCH"
)

// AppleCareCoverageAttributes : /v1/orgDevices/{id}/appleCareCoverage
type AppleCareCoverageAttributes struct {
	Status                 string `json:"status,omitempty"`      // AppleCareCoverageStatus: ACTIVE | INACTIVE
	PaymentType            string `json:"paymentType,omitempty"` // AppleCareCoveragePaymentType: ABE_SUBSCRIPTION | PAID_UP_FRONT | SUBSCRIPTION | NONE
	Description            string `json:"description,omitempty"`
	AgreementNumber        string `json:"agreementNumber"`
	StartDateTime          string `json:"startDateTime"`
	EndDateTime            string `json:"endDateTime"`
	ContractCancelDateTime string `json:"contractCancelDateTime"`
	IsCanceled             bool   `json:"isCanceled"`
	IsRenewable            bool   `json:"isRenewable"`
}

// ActivityAttributes : /v1/orgDeviceActivities
type ActivityAttributes struct {
	Status            string `json:"status"`
	SubStatus         string `json:"subStatus"`
	CreatedDateTime   string `json:"createdDateTime"`
	CompletedDateTime string `json:"completedDateTime,omitempty"`
	DownloadURL       string `json:"downloadUrl,omitempty"`
}

// ActivityTypeMetadata is additional metadata for an orgDeviceActivity, used
// by the device management service migration activity types (API 2.3+).
type ActivityTypeMetadata struct {
	// MdmMigrationDeadlineDateTime is the ISO 8601 deadline by which devices
	// must complete their migration (at most 90 days in the future).
	MdmMigrationDeadlineDateTime string `json:"mdmMigrationDeadlineDateTime,omitempty"`
}

// --- Device Management Services (devices enrolled in Apple MDM) ---

// MdmDevice is a device enrolled in Apple's built-in device management
// service, listed via GET /v1/mdmDevices.
type MdmDevice = applebusiness.ResourceObject[MdmDeviceAttributes]

// MdmDeviceAttributes holds the attributes of an MdmDevice (/v1/mdmDevices).
type MdmDeviceAttributes struct {
	DeviceName     string `json:"deviceName,omitempty"`
	EnrolledUserID string `json:"enrolledUserId,omitempty"`
	ProductFamily  string `json:"productFamily,omitempty"`
	SerialNumber   string `json:"serialNumber,omitempty"`
}

// MdmDeviceDetail holds the detailed information of a device enrolled in
// Apple's built-in device management service (GET /v1/mdmDevices/{id}/details).
type MdmDeviceDetail = applebusiness.ResourceObject[MdmDeviceDetailAttributes]

// MdmDeviceDetailAttributes holds the attributes of an MdmDeviceDetail.
type MdmDeviceDetailAttributes struct {
	DeviceName           string   `json:"deviceName,omitempty"`
	DeviceModel          string   `json:"deviceModel,omitempty"`
	SerialNumber         string   `json:"serialNumber,omitempty"`
	OSVersion            string   `json:"osVersion,omitempty"`
	Platform             string   `json:"platform,omitempty"`
	IMEI                 []string `json:"imei,omitempty"`
	MEID                 []string `json:"meid,omitempty"`
	WifiMacAddress       string   `json:"wifiMacAddress,omitempty"`
	BluetoothMacAddress  string   `json:"bluetoothMacAddress,omitempty"`
	EthernetMacAddress   string   `json:"ethernetMacAddress,omitempty"`
	StorageFreeCapacity  int64    `json:"storageFreeCapacity,omitempty"`
	StorageTotalCapacity int64    `json:"storageTotalCapacity,omitempty"`
	DeviceEraseStatus    string   `json:"deviceEraseStatus,omitempty"` // NOT_ERASED | ERASED
	DeviceLockStatus     string   `json:"deviceLockStatus,omitempty"`  // LOCKED | UNLOCKED
	LostModeStatus       string   `json:"lostModeStatus,omitempty"`    // ENABLED | DISABLED
	IsFileVaultEnabled   bool     `json:"isFileVaultEnabled,omitempty"`
	IsFirewallEnabled    bool     `json:"isFirewallEnabled,omitempty"`
	LastCheckInDateTime  string   `json:"lastCheckInDateTime,omitempty"`
}

// Device-state enumeration values (DeviceEraseStatus / DeviceLockStatus / LostModeStatus).
const (
	EraseStatusNotErased = "NOT_ERASED"
	EraseStatusErased    = "ERASED"
	LockStatusLocked     = "LOCKED"
	LockStatusUnlocked   = "UNLOCKED"
	LostModeEnabled      = "ENABLED"
	LostModeDisabled     = "DISABLED"
)
