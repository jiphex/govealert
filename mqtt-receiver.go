package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	//"encoding/json"

	"code.google.com/p/goprotobuf/proto"
	mqtt "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	. "repos.bytemark.co.uk/govealert/mauve"
)

func mqttDisconnect(client *mqtt.MqttClient, reason error) {
	log.Fatalf("Lost MQTT Connection because: %s", reason)
}

func alertTopic(al *Alert, source string) string {
	return fmt.Sprintf("%s/%s/%s", source, *al.Subject, *al.Id)
}

func parseAlertTopic(baseTopic string, topic string) (source string, subject string, id string) {
	lBase = len(strings.Split(baseTopic, "/")) // todo: deal with leading/trailing slashes
	parts := strings.SplitN(topic, "/", lBase+3)
	return parts[-3], parts[-2], parts[-1]
}

func convertStreaming(inc <-chan MQTT.Message, out chan<- *AlertUpdate) {
	for {
		m := <-inc
		log.Printf("Got %s", m)
		alert := proto.Unmarshal(m.Payload)

		out <- alert
	}
}

func convertAlerts(messageChannel <-chan MQTT.Message) chan<- *AlertUpdate {
	ms := make(chan *Alert)
	go convertStreaming(messageChannel, ms)
	return ms
}

func DialMQTT(broker string, topicBase string) (<-chan *Alert, error) {
	incomingAlerts := make(chan mqtt.Message)
	mqttOpts := mqtt.NewClientOptions().AddBroker(broker).SetClientId("govealert-mqtt-receiver").SetCleanSession(true).SetOnConnectionLost(mqttDisconnect)
	client := mqtt.NewClient(mqttOpts)
	mqttOpt.SetDefaultPublishHandler(func(client *mqtt.MqttClient, msg mqtt.Message) {
		incomingAlerts <- msg
	})
	if _, err := client.Start(); err != nil {
		log.Printf("Failed to connect to MQTT Broker: %s - %s", broker, err)
		return nil, err
	} else {
		log.Printf("Connected to Broker")
	}
	return convertAlerts(incomingAlerts)
}

func DialMauve(source string, replace bool, host string, queue <-chan *Alert, receipt chan<- uint64) {
	// This connects to Mauve over UDP and then waits on it's channel,
	// any Alert that gets written to the channel will get wrapped in
	// an AlertUpdate and then sent to the Mauve server
	addr, err := net.ResolveUDPAddr("udp", host)
	if err != nil {
		log.Fatal("Cannot resolve mauvealert server: %s", addr)
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Fatal("Failed to connect to mauve: %s", addr)
	}
	defer conn.Close() // Just make sure that the connection gets flushed
	//log.Printf("dialing...")
	for {
		al := <-queue
		up := CreateUpdate(source, replace, al)
		mu, err := proto.Marshal(up)
		if err != nil {
			log.Fatalf("Failed to marshal an alertUpdate: %s", err)
		}
		//log.Printf("Sent: %s", up.String())
		if _, err := conn.Write(mu); err != nil {
			log.Fatalf("Failed to send message: %s", err)
		} else {
			receipt <- *up.TransmissionId
		}
	}
}

func main() {
	hostname, _ := os.Hostname()
	mauvealert := flag.String("mauve", "alert.bytemark.co.uk:32741", "Mauve (or MQTT) Server to dial")
	//heartbeat := flag.Bool("heartbeat", false, "Don't do normal operation, just send a 10 minute heartbeat")
	//cancel := flag.Bool("cancel", false, "When specified with -heartbeat, cancels the heartbeat (via suppress+raise, clear)")
	mqttBroker := flag.String("mqtt-broker", "tcp://localhost:1883", "The MQTT Broker to connect to")
	mqttTopic := flag.String("mqtt-base", "/govealert", "Base topic for MQTT transport packets")
	flag.Parse()

	msend := make(chan *Alert, 5)
	// This is just a channel we wait on to make sure we only send one alert at once
	receipt := make(chan uint64)
	if *transport == "mqtt" {
		go DialMQTT(*source, *mqttBroker, *mqttTopic, msend, receipt)
	} else if *transport == "protobuf" {
		go DialMauve(*source, *replace, *mauvealert, msend, receipt)
	} else {
		log.Fatalf("Invalid alert transport: %s", *transport)
	}
	if *heartbeat {
		hbsumm := fmt.Sprintf("heartbeat failed for %s", hostname)
		hbdetail := fmt.Sprintf("The govealert heartbeat wasn't sent for the host %s.", hostname)
		hbid := "heartbeat"
		hbraise := "+10m"
		hbclear := "now"
		if *cancel {
			// Cancel a heartbeat alert by sending: suppressed raise, clear (experimental)
			//log.Printf("Cancelling alert heartbeat")
			supraise := "now"
			suptime := "+5m"
			sup := CreateAlert(hbid, supraise, hbclear, hostname, hbsumm, hbdetail, suptime)
			msend <- sup
			<-receipt
			clr := CreateAlert(hbid, hbclear, supraise, hostname, hbsumm, hbdetail, hbclear)
			msend <- clr
			<-receipt
		} else {
			// 	Send a hearbeat alert (clear now, raise in 10 minutes - meant to be called every N where N < 5 minutes)
			al := CreateAlert(hbid, hbraise, hbclear, hostname, hbsumm, hbdetail, hbclear)
			msend <- al
			<-receipt
		}
	} else {
		custom := CreateAlert(*id, *raise, *clear, *subject, *summary, *detail, *suppress)
		msend <- custom
		<-receipt
	}
}
