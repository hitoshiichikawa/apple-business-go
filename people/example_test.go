package people_test

import (
	"context"
	"fmt"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
	"github.com/hitoshiichikawa/apple-business-go/people"
)

// ユーザー一覧を取得する（ページングは自動）。
func ExampleService_ListUsers() {
	var c *applebusiness.Client // 実際には applebusiness.NewClient で生成する
	svc := people.New(c)
	users, err := svc.ListUsers(context.Background(), nil)
	if err != nil {
		return
	}
	for _, u := range users {
		fmt.Println(u.ID, u.Attributes.Email, u.Attributes.Status)
	}
}

// ユーザーグループの一覧と、各グループの所属ユーザーID を取得する。
func ExampleService_ListUserGroups() {
	var c *applebusiness.Client // 実際には applebusiness.NewClient で生成する
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
