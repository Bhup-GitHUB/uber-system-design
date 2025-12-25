package models

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
	CarType   string    `json:"car_type"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SearchRequest struct {
	Location  Location `json:"location"`
	Radius    float64  `json:"radius"`
	IndexType string   `json:"index_type,omitempty"`
}

type SearchResponse struct {
	Drivers   []DriverWithDistance `json:"drivers"`
	Count     int                  `json:"count"`
	Duration  string               `json:"duration"`
	IndexType string               `json:"index_type"`
}

type DriverWithDistance struct {
	Driver   Driver  `json:"driver"`
	Distance float64 `json:"distance"`
}

type UpdateLocationRequest struct {
	DriverID string  `json:"driver_id"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
}

type UpdateStatusRequest struct {
	DriverID string `json:"driver_id"`
	Status   string `json:"status"`
}

type ComparisonResult struct {
	QuadTree map[string]interface{} `json:"quadtree"`
	Grid     map[string]interface{} `json:"grid"`
	Redis    map[string]interface{} `json:"redis,omitempty"`
}

