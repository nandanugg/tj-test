package config

import "os"

type Config struct {
	PostgresDSN  string
	RabbitMQURL  string
	MQTTBroker   string
	MQTTClientID string
	HTTPPort     string
}

func Load() *Config {
	return &Config{
		PostgresDSN:  getEnv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/fleet?sslmode=disable"),
		RabbitMQURL:  getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		MQTTBroker:   getEnv("MQTT_BROKER", "tcp://localhost:1883"),
		MQTTClientID: getEnv("MQTT_CLIENT_ID", "fleet-server"),
		HTTPPort:     getEnv("HTTP_PORT", "8080"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
