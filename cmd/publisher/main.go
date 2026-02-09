package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type locationMessage struct {
	VehicleID string  `json:"vehicle_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timestamp int64   `json:"timestamp"`
}

const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomVehicleID() string {
	letter := string(charset[rand.Intn(26)])
	digits := fmt.Sprintf("%04d", rand.Intn(10000))
	suffix := string([]byte{charset[rand.Intn(26)], charset[rand.Intn(26)], charset[rand.Intn(26)]})
	return letter + digits + suffix
}

func randomLat() float64 {
	return -90 + rand.Float64()*180
}

func randomLon() float64 {
	return -180 + rand.Float64()*360
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <interval_seconds>\n", os.Args[0])
		os.Exit(1)
	}

	intervalSec, err := strconv.Atoi(os.Args[1])
	if err != nil || intervalSec <= 0 {
		fmt.Fprintf(os.Stderr, "error: interval must be a positive integer\n")
		os.Exit(1)
	}

	broker := "tcp://localhost:1883"
	if v := os.Getenv("MQTT_BROKER"); v != "" {
		broker = v
	}

	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID("fleet-mock-publisher")

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("mqtt connect: %v", token.Error())
	}
	defer client.Disconnect(250)

	vehiclePool := make([]string, 5)
	for i := range vehiclePool {
		vehiclePool[i] = randomVehicleID()
	}

	log.Printf("connected to %s, publishing every %ds...", broker, intervalSec)
	log.Printf("vehicle pool: %v", vehiclePool)

	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		vid := vehiclePool[rand.Intn(len(vehiclePool))]

		var lat, lon float64
		// 30% chance to send near geofence point (-6.2088, 106.8456)
		if rand.Float64() < 0.3 {
			lat = -6.2088 + (rand.Float64()-0.5)*0.0005 // ~50m drift
			lon = 106.8456 + (rand.Float64()-0.5)*0.0005
		} else {
			lat = randomLat()
			lon = randomLon()
		}

		msg := locationMessage{
			VehicleID: vid,
			Latitude:  lat,
			Longitude: lon,
			Timestamp: time.Now().Unix(),
		}

		payload, _ := json.Marshal(msg)
		topic := fmt.Sprintf("/fleet/vehicle/%s/location", vid)

		token := client.Publish(topic, 1, false, payload)
		token.Wait()

		log.Printf("published to %s: %s", topic, payload)
	}
}
