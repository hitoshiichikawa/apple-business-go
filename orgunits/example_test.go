package orgunits_test

import (
	"context"
	"fmt"

	"github.com/hitoshiichikawa/apple-business-go/applebusiness"
	"github.com/hitoshiichikawa/apple-business-go/orgunits"
)

// List organizational units and the member user IDs of each unit.
func ExampleService_List() {
	var c *applebusiness.Client // in practice, create this with applebusiness.NewClient
	svc := orgunits.New(c)
	units, err := svc.List(context.Background(), nil)
	if err != nil {
		return
	}
	for _, ou := range units {
		members, err := svc.Members(context.Background(), ou.ID)
		if err != nil {
			continue
		}
		fmt.Println(ou.ID, ou.Attributes.Name, len(members))
	}
}
