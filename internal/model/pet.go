package model

type Pet struct {
	ID       int64     `json:"id"`
	Category *Category `json:"category"`
	Name     string    `json:"name"`
	Status   string    `json:"status"`
	ImageURL string    `json:"photoUrls"`
	Tags     []*Tag    `json:"tags"`
}
type Tag struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
type Category struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
