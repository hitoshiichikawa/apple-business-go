package people

import "github.com/hitoshiichikawa/apple-business-go/applebusiness"

type (
	User      = applebusiness.ResourceObject[UserAttributes]
	UserGroup = applebusiness.ResourceObject[UserGroupAttributes]
)

type PhoneNumber struct {
	PhoneNumber string `json:"phoneNumber"`
	Type        string `json:"type"`
}

type RoleOU struct {
	OUID     string `json:"ouId"`
	RoleName string `json:"roleName"`
}

// UserAttributes : /v1/users
type UserAttributes struct {
	ManagedAppleAccount string        `json:"managedAppleAccount,omitempty"`
	Email               string        `json:"email,omitempty"`
	FirstName           string        `json:"firstName,omitempty"`
	MiddleName          string        `json:"middleName,omitempty"`
	LastName            string        `json:"lastName,omitempty"`
	JobTitle            string        `json:"jobTitle,omitempty"`
	Department          string        `json:"department,omitempty"`
	Division            string        `json:"division,omitempty"`
	CostCenter          string        `json:"costCenter,omitempty"`
	EmployeeNumber      string        `json:"employeeNumber,omitempty"`
	IsExternalUser      bool          `json:"isExternalUser,omitempty"`
	Status              string        `json:"status,omitempty"`
	PhoneNumbers        []PhoneNumber `json:"phoneNumbers,omitempty"`
	RoleOUList          []RoleOU      `json:"roleOuList,omitempty"`
	StartDateTime       string        `json:"startDateTime,omitempty"`
	CreatedDateTime     string        `json:"createdDateTime,omitempty"`
	UpdatedDateTime     string        `json:"updatedDateTime,omitempty"`
}

// UserGroupAttributes : /v1/userGroups
type UserGroupAttributes struct {
	Name             string `json:"name,omitempty"`
	Type             string `json:"type,omitempty"` // UserGroupType: STANDARD | SMART
	OUID             string `json:"ouId,omitempty"`
	Status           string `json:"status,omitempty"` // UserGroupStatus: ACTIVE
	TotalMemberCount int    `json:"totalMemberCount,omitempty"`
	CreatedDateTime  string `json:"createdDateTime,omitempty"`
	UpdatedDateTime  string `json:"updatedDateTime,omitempty"`
}

// UserGroupType の値。
const (
	UserGroupStandard = "STANDARD"
	UserGroupSmart    = "SMART"
)

// UserStatus の値。
const (
	UserStatusNew              = "NEW"
	UserStatusReleased         = "RELEASED"
	UserStatusActive           = "ACTIVE"
	UserStatusDeactivated      = "DEACTIVATED"
	UserStatusLocked           = "LOCKED"
	UserStatusLockedSharedIPad = "LOCKED_FOR_SHARED_IPAD"
)
