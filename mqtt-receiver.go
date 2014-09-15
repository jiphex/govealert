package main

import (
	"encoding/json"
	"flag"
	"log"
	"net"
	"os"
	"time"
	"fmt"

	"code.google.com/p/goprotobuf/proto"
	mqtt "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	. "repos.bytemark.co.uk/govealert/mauve"
)

func mqttDisconnect(client *mqtt.MqttClient, reason error) {
	log.Fatalf("Lost MQTT Connection because: %s", reason)
}

func convertStreaming(baseTopic string, inc <-chan mqtt.Message, out chan<- *AlertUpdate) {
	for {
		m := <-inc
		log.Printf("Got %s", m)
		alert := new(Alert)
		err := proto.Unmarshal(m.Payload(), alert)
		source, _, _ := ParseAlertTopic(baseTopic, m.Topic())
		up := CreateUpdate(source, false, alert)
		if err != nil {
			log.Printf("Skipping packet that failed to unmarshal")
		} else {
			out <- up
		}
	}
}

func convertAlerts(baseTopic string, incMessages <-chan mqtt.Message) <-chan *AlertUpdate {
	ms := make(chan *AlertUpdate)
	go convertStreaming(baseTopic, incMessages, ms)
	return ms
}

func mqttStatusPacket() []byte {
	hostname, _ := os.Hostname()
	now := time.Now().Unix()
	status := map[string]string{
		"hostname": hostname,
		"now":      string(now),
	}
	json, _ := json.Marshal(status)
	return json
}

func mqttHeartbeat(topicBase string, secs time.Duration, client mqtt.MqttClient) {
	for {
		hostname, _ := os.Hostname()
		s := mqttStatusPacket()
		publishTopic := fmt.Sprintf("%s/$heartbeat/%s", topicBase, hostname)
		client.Publish(mqtt.QOS_ONE, publishTopic, s)
		time.Sleep(secs)
	}
}


func dialMauve(replace bool, host string, queue <-chan *AlertUpdate) {
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
		up := <-queue
		mu, err := proto.Marshal(up)
		if err != nil {
			log.Fatalf("Failed to marshal an alertUpdate: %s", err)
		}
		//log.Printf("Sent: %s", up.String())
		if _, err := conn.Write(mu); err != nil {
			log.Fatalf("Failed to send message: %s", err)
		}
	}
}

/*
So what we want to do here is to sit and listen on the MQTT channel
provided and receive MQTTMessages containing Alerts.

Every Alert needs to be wrapped into an AlertUpdate, and then passed to
Mauve.
*/

func main() {
	mauvealert := flag.String("mauve", "alert.bytemark.co.uk:32741", "Mauve (or MQTT) Server to dial")
	//heartbeat := flag.Bool("heartbeat", false, "Don't do normal operation, just send a 10 minute heartbeat")
	//cancel := flag.Bool("cancel", false, "When specified with -heartbeat, cancels the heartbeat (via suppress+raise, clear)")
	mqttBroker := flag.String("mqtt-broker", "tcp://localhost:1883", "The MQTT Broker to connect to")
	mqttTopic := flag.String("mqtt-base", "/govealert", "Base topic for MQTT transport packets")
	flag.Parse()

	msend := make(chan *AlertUpdate, 50)    // the channel we'll dump AlertUpdate packets destined for Mauve into
	go dialMauve(false, *mauvealert, msend) // this goroutine will send any packets on the msend channel into mauve

	incomingAlerts := make(chan mqtt.Message)
	mqttOpts := mqtt.NewClientOptions().AddBroker(*mqttBroker).SetClientId("govealert-mqtt-receiver").SetCleanSession(true).SetOnConnectionLost(mqttDisconnect)
	filter,_ := mqtt.NewTopicFilter(fmt.Sprintf("%s/+/+/+",*mqttTopic),byte(1))
	client := mqtt.NewClient(mqttOpts)
    msgHandler := func(client *mqtt.MqttClient, msg mqtt.Message) {
		log.Printf("Packet on %s", msg.Topic())
		incomingAlerts <- msg
	}
	if _, err := client.Start(); err != nil {
		log.Fatalf("Failed to connect to MQTT Broker: %s - %s", *mqttBroker, err)
	} else {
		log.Printf("Connected to Broker")
		if _,err := client.StartSubscription(msgHandler,filter); err != nil {
			log.Printf("Failed to subscribe: %s", err)
		}
	}
    incAlertUpdate := convertAlerts(*mqttTopic, incomingAlerts)
    var inc *AlertUpdate
	for {
		inc = <-incAlertUpdate
		msend <- inc
	}
}
