package people_test

import (
	"context"
	"fmt"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
	"github.com/hitoshiichikawa/apple-business-go/people"
)

// List all users (pagination handled automatically).
func ExampleService_ListUsers() {
	var c *applebusiness.Client // in practice, create this with applebusiness.NewClient
	svc := people.New(c)
	users, err := svc.ListUsers(context.Background(), nil)
	if err != nil {
		return
	}
	for _, u := range users {
		fmt.Println(u.ID, u.Attributes.Email, u.Attributes.Status)
	}
}

// List user groups and the member user IDs of each group.
func ExampleService_ListUserGroups() {
	var c *applebusiness.Client // in practice, create this with applebusiness.NewClient
	svc := people.New(c)
	groups, err := svc.ListUserGroups(context.Background(), nil)
	if err != nil {
		return
	}
	for _, g := range groups {
		members, err := svc.GroupMembers(context.Background(), g.ID)
		if err != nil {
			continue
		}
		fmt.Println(g.ID, g.Attributes.Name, len(members))
	}
}
