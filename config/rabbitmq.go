package config

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func NewRabbitMQ(cfg *Config) (*amqp.Connection, error) {
	conn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq connect: %w", err)
	}
	return conn, nil
}
