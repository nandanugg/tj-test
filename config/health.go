package config

import (
	"database/sql"
	"net/http"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
)

type HealthChecker struct {
	db       *sql.DB
	amqpConn *amqp.Connection
	mqtt     mqtt.Client
}

func NewHealthChecker(db *sql.DB, amqpConn *amqp.Connection, mqttClient mqtt.Client) *HealthChecker {
	return &HealthChecker{db: db, amqpConn: amqpConn, mqtt: mqttClient}
}

func (h *HealthChecker) Register(r *gin.Engine) {
	r.GET("/healthz", h.Handle)
}

func (h *HealthChecker) Handle(c *gin.Context) {
	status := http.StatusOK
	deps := gin.H{}

	if err := h.db.PingContext(c.Request.Context()); err != nil {
		deps["postgres"] = gin.H{"status": "down", "error": err.Error()}
		status = http.StatusServiceUnavailable
	} else {
		deps["postgres"] = gin.H{"status": "up"}
	}

	if h.amqpConn.IsClosed() {
		deps["rabbitmq"] = gin.H{"status": "down", "error": "connection closed"}
		status = http.StatusServiceUnavailable
	} else {
		deps["rabbitmq"] = gin.H{"status": "up"}
	}

	if !h.mqtt.IsConnected() {
		deps["mqtt"] = gin.H{"status": "down", "error": "not connected"}
		status = http.StatusServiceUnavailable
	} else {
		deps["mqtt"] = gin.H{"status": "up"}
	}

	overall := "healthy"
	if status != http.StatusOK {
		overall = "unhealthy"
	}

	c.JSON(status, gin.H{
		"status":       overall,
		"dependencies": deps,
	})
}
