package http

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nandanugg/tj-test/module/core/domain"
)

type locationService interface {
	GetLatest(ctx context.Context, vehicleID string) (*domain.VehicleLocation, error)
	GetHistory(ctx context.Context, query *domain.HistoryQuery) ([]domain.VehicleLocation, error)
	GetAllVehicles(ctx context.Context) ([]domain.Vehicle, error)
}

type locationResponse struct {
	VehicleID string  `json:"vehicle_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timestamp int64   `json:"timestamp"`
}

type VehicleHandler struct {
	locationSvc locationService
}

func NewVehicleHandler(locationSvc locationService) *VehicleHandler {
	return &VehicleHandler{locationSvc: locationSvc}
}

func (h *VehicleHandler) Register(r *gin.RouterGroup) {
	r.GET("/vehicles", h.GetAllVehicles)
	r.GET("/vehicles/:vehicle_id/location", h.GetLatestLocation)
	r.GET("/vehicles/:vehicle_id/history", h.GetHistory)
}

func (h *VehicleHandler) GetAllVehicles(c *gin.Context) {
	vehicles, err := h.locationSvc.GetAllVehicles(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch vehicles"})
		return
	}

	c.JSON(http.StatusOK, vehicles)
}

func (h *VehicleHandler) GetLatestLocation(c *gin.Context) {
	vehicleID := c.Param("vehicle_id")

	vl, err := h.locationSvc.GetLatest(c.Request.Context(), vehicleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "vehicle not found"})
		return
	}

	c.JSON(http.StatusOK, toLocationResponse(vl))
}

func (h *VehicleHandler) GetHistory(c *gin.Context) {
	vehicleID := c.Param("vehicle_id")

	start, err := strconv.ParseInt(c.Query("start"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start parameter"})
		return
	}

	end, err := strconv.ParseInt(c.Query("end"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end parameter"})
		return
	}

	query := &domain.HistoryQuery{
		VehicleID: vehicleID,
		Start:     time.Unix(start, 0),
		End:       time.Unix(end, 0),
	}

	locations, err := h.locationSvc.GetHistory(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch history"})
		return
	}

	results := make([]locationResponse, len(locations))
	for i, vl := range locations {
		results[i] = toLocationResponse(&vl)
	}
	c.JSON(http.StatusOK, results)
}

func toLocationResponse(vl *domain.VehicleLocation) locationResponse {
	return locationResponse{
		VehicleID: vl.VehicleID,
		Latitude:  vl.Location.Lat,
		Longitude: vl.Location.Lon,
		Timestamp: vl.Location.Timestamp.Unix(),
	}
}
