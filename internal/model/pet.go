package model

type Pet struct {
	ID       int64     `json:"id"`
	Name     string    `json:"name"`
	Category *Category `json:"category,omitempty"` // omitempty для nil категории
	Status   string    `json:"status"`
	ImageURL string    `json:"photoUrls"` // Изменили имя поля для соответствия API
	Tags     []*Tag    `json:"tags,omitempty"`
}

type Category struct {
	ID   int64  `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type Tag struct {
	ID   int64  `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type StatusRequest struct {
	Status string `json:"status"`
}
