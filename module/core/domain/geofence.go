package domain

type GeoPoint struct {
	Lat    float64 `json:"latitude"`
	Lon    float64 `json:"longitude"`
	Radius float64 `json:"radius"`
}

type GeofenceEventType string

const (
	GeofenceEntry GeofenceEventType = "geofence_entry"
	GeofenceExit  GeofenceEventType = "geofence_exit"
)

type GeofenceAlert struct {
	VehicleID string            `json:"vehicle_id"`
	Event     GeofenceEventType `json:"event"`
	Location  Location          `json:"location"`
	Timestamp int64             `json:"timestamp"`
}
