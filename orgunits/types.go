package orgunits

import "github.com/hitoshiichikawa/apple-business-go/applebusiness"

// OrganizationalUnit is an organizational-unit resource (/v1/organizationalUnits).
type OrganizationalUnit = applebusiness.ResourceObject[OrganizationalUnitAttributes]

// OrganizationalUnitAttributes : /v1/organizationalUnits. Organizational units
// form the org-structure layer that users and user groups reference
// (people.RoleOU.OUID and people.UserGroupAttributes.OUID).
type OrganizationalUnitAttributes struct {
	Name            string `json:"name,omitempty"`
	Description     string `json:"description,omitempty"`
	CreatedDateTime string `json:"createdDateTime,omitempty"`
	UpdatedDateTime string `json:"updatedDateTime,omitempty"`
}
