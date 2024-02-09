package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
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

// Command constants
const (
	CommandPowerOn  uint16 = 0x2B2
	CommandPowerOff uint16 = 0x2B2
	CommandSetRange uint16 = 0x2B3
	CommandSetGain  uint16 = 0x2B4
)

// Global UDP connection for sending control commands to the radar
var controlConn *net.UDPConn

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

func handleMQTTMessage(payload []byte) {
	var cmd RadarCommand
	if err := json.Unmarshal(payload, &cmd); err != nil {
		log.Printf("Error unmarshalling command: %v", err)
		return
	}

	switch cmd.Action {
	case "power_on":
		sendPowerCommand(true)
	case "power_off":
		sendPowerCommand(false)
	case "set_range":
		sendSetRangeCommand(cmd.Value)
	case "set_gain":
		sendSetGainCommand(cmd.Manual, cmd.Value)

	default:
		log.Printf("Unknown command action: %s", cmd.Action)
	}
}

func sendPowerCommand(on bool) {
	// Sending power on/off command
	data := uint16(0)
	if on {
		data = 2 // power on
	} else {
		data = 1 // power off
	}
	sendControlCommand(CommandPowerOn, data)
	fmt.Printf("Power command sent: %v\n", on)
}

func sendSetRangeCommand(rangeNm float64) {
	// Convert nautical miles to the radar's range unit
	rangeVal := uint16(rangeNm * 100) // Example conversion
	sendControlCommand(CommandSetRange, rangeVal)
	fmt.Printf("Set range command sent: %f nm\n", rangeNm)
}

func sendSetGainCommand(manual bool, value float64) {
	// Convert gain setting to radar's protocol
	gainVal := uint16(value) //Conversion for manual gain setting
	if !manual {
		gainVal = 0 // auto gain
	}
	sendControlCommand(CommandSetGain, gainVal)
	fmt.Printf("Set gain command sent: Manual=%v, Value=%f\n", manual, value)
}

func sendControlCommand(commandType uint16, value uint16) {
	// Serialize command according to the radar's protocol
	var msg []byte
	msg = make([]byte, 6)
	binary.BigEndian.PutUint16(msg[0:2], commandType)
	binary.BigEndian.PutUint16(msg[2:4], 2) // Length of the data
	binary.BigEndian.PutUint16(msg[4:6], value)

	_, err := controlConn.Write(msg)
	if err != nil {
		log.Printf("Failed to send control command: %v", err)
	}
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
