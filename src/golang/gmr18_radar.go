package main

import (
	"fmt"
	"log"
	"net"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type GarminSample struct {
	Angle   int
	Range   int
	Samples []byte
}

type RadarCommand struct {
	Action string  `json:"action"`
	Value  float64 `json:"value,omitempty"`
	Manual bool    `json:"manual,omitempty"`
	On     bool    `json:"on,omitempty"`

func main() {
	localAddress := "0.0.0.0"
	remoteAddress := "172.16.2.0"
	multicastAddress := "239.254.2.0"
	mqttBroker := "tcp://mqtt.yourbroker.com:1883" // Broker address

	controlAddr, err := net.ResolveUDPAddr("udp", remoteAddress+":50101")
	if err != nil {
		log.Fatalf("Failed to resolve remote address: %v", err)
	}
	controlConn, err := net.DialUDP("udp", nil, controlAddr)
	if err != nil {
		log.Fatalf("Failed to dial UDP for control: %v", err)
	}
	defer controlConn.Close()

	multicastAddr, err := net.ResolveUDPAddr("udp", multicastAddress+":50100")
	if err != nil {
		log.Fatalf("Failed to resolve multicast address: %v", err)
	}
	localAddr, err := net.ResolveUDPAddr("udp", localAddress+":0")
	if err != nil {
		log.Fatalf("Failed to resolve local address for multicast: %v", err)
	}
	multicastConn, err := net.ListenMulticastUDP("udp", nil, multicastAddr)
	if err != nil {
		log.Fatalf("Failed to listen on multicast address: %v", err)
	}
	multicastConn.SetReadBuffer(1024 * 1024) // Buffer size
	defer multicastConn.Close()

	opts := mqtt.NewClientOptions().AddBroker(mqttBroker)
	mqttClient := mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	}

	mqttTopic := "garmin/gmr18radar"

	messagePubHandler := func(client mqtt.Client, msg mqtt.Message) {
		fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
	}

	if token := mqttClient.Subscribe("garmin/radar/command", 0, messagePubHandler); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to subscribe: %v", token.Error())
	}

	go func() {
		for {
			handleIncomingData(multicastConn, controlConn, mqttClient, mqttTopic)
		}
	}()

	select {} // Keep the main goroutine alive
}

func handleIncomingData(multicastConn *net.UDPConn, controlConn *net.UDPConn, mqttClient mqtt.Client, mqttTopic string) {
	buffer := make([]byte, 4096)
	length, _, err := multicastConn.ReadFromUDP(buffer)
	if err != nil {
		log.Printf("Error reading from multicast: %v", err)
		return
	}

	fmt.Printf("Received %d bytes of data\n", length)
	// Process the data here and publish to MQTT

	// Publishing a message
	text := fmt.Sprintf("Sample message at %s", time.Now().Format(time.RFC3339))
	mqttClient.Publish(mqttTopic, 0, false, text)
}
