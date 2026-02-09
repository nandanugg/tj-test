package domain

import "time"

type Location struct {
	Lat       float64   `json:"latitude"`
	Lon       float64   `json:"longitude"`
	Timestamp time.Time `json:"timestamp"`
}

type VehicleLocation struct {
	VehicleID string   `json:"vehicle_id"`
	Location  Location `json:"location"`
}

type HistoryQuery struct {
	VehicleID string
	Start     time.Time
	End       time.Time
}
