package applebusiness

// JSON:API envelope types. Each entity's attributes are carried by the type parameter A.

// Data is a JSON:API resource identifier ({type,id}).
type Data struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// Links holds self / next and similar URLs. next is the absolute URL of the next page
// (with the cursor embedded).
type Links struct {
	Self    string `json:"self,omitempty"`
	Next    string `json:"next,omitempty"`
	Related string `json:"related,omitempty"`
}

type Paging struct {
	NextCursor string `json:"nextCursor,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

type Meta struct {
	Paging Paging `json:"paging,omitempty"`
}

// ResourceObject is a single resource with typed attributes.
type ResourceObject[A any] struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	Attributes A      `json:"attributes"`
	Links      Links  `json:"links,omitempty"`
}

type ListResponse[A any] struct {
	Data  []ResourceObject[A] `json:"data"`
	Links Links               `json:"links"`
	Meta  Meta                `json:"meta"`
}

type SingleResponse[A any] struct {
	Data  ResourceObject[A] `json:"data"`
	Links Links             `json:"links"`
}

type RelationshipResponse struct {
	Data  []Data `json:"data"`
	Links Links  `json:"links"`
	Meta  Meta   `json:"meta"`
}
