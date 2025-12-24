package common

import "time"

type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type Driver struct {
	ID        string    `json:"id"`
	Location  Location  `json:"location"`
	Status    string    `json:"status"`
	Rating    float64   `json:"rating"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SearchRequest struct {
	Location Location `json:"location"`
	Radius   float64  `json:"radius"`
}

type SearchResponse struct {
	Drivers  []DriverWithDistance `json:"drivers"`
	Count    int                  `json:"count"`
	Duration string               `json:"duration"`
}

type DriverWithDistance struct {
	Driver   Driver  `json:"driver"`
	Distance float64 `json:"distance"`
}

