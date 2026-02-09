package config

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func NewMQTT(cfg *Config) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions().
		AddBroker(cfg.MQTTBroker).
		SetClientID(cfg.MQTTClientID)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("mqtt connect: %w", token.Error())
	}
	return client, nil
}
