package applebusiness

// JSON:API のエンベロープ型。各エンティティの属性は型パラメータ A で受ける。

// Data は JSON:API のリソース識別子 ({type,id})。
type Data struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// Links は self / next など。next は次ページの絶対URL（カーソル埋め込み）。
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

// ResourceObject は型付き属性を持つ単一リソース。
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
