package model

import "time"

type Order struct {
	ID       int64     `json:"id"`
	UserID   int64     `json:"userId"`
	PetID    int64     `json:"petId"`
	Quantity int       `json:"quantity"`
	ShipDate time.Time `json:"shipDate"`
	Status   string    `json:"status"`
	Complete bool      `json:"complete"`
}
