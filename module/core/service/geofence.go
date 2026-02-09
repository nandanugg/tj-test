package service

import (
	"context"
	"math"

	"github.com/nandanugg/tj-test/module/core/domain"
	"github.com/nandanugg/tj-test/module/core/internal/repository/publisher"
)

const earthRadiusMeters = 6371000

type GeofenceService struct {
	publisher publisher.GeofencePublisher
	geofences []domain.GeoPoint
}

func NewGeofenceService(pub publisher.GeofencePublisher, geofences []domain.GeoPoint) *GeofenceService {
	return &GeofenceService{
		publisher: pub,
		geofences: geofences,
	}
}

func (s *GeofenceService) CheckAndAlert(ctx context.Context, vl *domain.VehicleLocation) error {
	for _, gf := range s.geofences {
		dist := haversine(vl.Location.Lat, vl.Location.Lon, gf.Lat, gf.Lon)
		if dist <= gf.Radius {
			alert := &domain.GeofenceAlert{
				VehicleID: vl.VehicleID,
				Event:     domain.GeofenceEntry,
				Location:  vl.Location,
				Timestamp: vl.Location.Timestamp.Unix(),
			}
			if err := s.publisher.PublishAlert(ctx, alert); err != nil {
				return err
			}
		}
	}
	return nil
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadiusMeters * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func toRad(deg float64) float64 {
	return deg * math.Pi / 180
}
