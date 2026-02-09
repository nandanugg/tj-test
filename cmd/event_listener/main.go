package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	amqp "github.com/rabbitmq/amqp091-go"
)

const queueName = "geofence_alerts"

func main() {
	url := "amqp://guest:guest@localhost:5672/"
	if v := os.Getenv("RABBITMQ_URL"); v != "" {
		url = v
	}

	conn, err := amqp.Dial(url)
	if err != nil {
		log.Fatalf("rabbitmq connect: %v", err)
	}
	defer func() { _ = conn.Close() }()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("rabbitmq channel: %v", err)
	}
	defer func() { _ = ch.Close() }()

	if err := ch.ExchangeDeclare("fleet.events", "fanout", true, false, false, false, nil); err != nil {
		log.Fatalf("declare exchange: %v", err)
	}

	if _, err := ch.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
		log.Fatalf("declare queue: %v", err)
	}

	if err := ch.QueueBind(queueName, "", "fleet.events", false, nil); err != nil {
		log.Fatalf("bind queue: %v", err)
	}

	msgs, err := ch.Consume(queueName, "", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("consume: %v", err)
	}

	log.Printf("consuming from queue '%s', waiting for geofence alerts...", queueName)

	go func() {
		for msg := range msgs {
			var alert struct {
				Event string `json:"event"`
			}
			if err := json.Unmarshal(msg.Body, &alert); err != nil {
				continue
			}
			fmt.Printf("[%s] %s\n", alert.Event, string(msg.Body))
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("shutting down")
}
