package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"
	"strconv"

	"code.google.com/p/goprotobuf/proto"
	mqtt "git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	. "repos.bytemark.co.uk/govealert/mauve"
)

func mqttDisconnect(client *mqtt.MqttClient, reason error) {
	log.Fatalf("Lost MQTT Connection because: %s", reason)
}

func unmarshalAlert(payload []byte) (*Alert,error) {
	alert := new(Alert)
	// first, try as protobuf
	err := proto.Unmarshal(payload,alert)
	if err != nil {
		// failed to unmarshal as proto, try as json
		alert = new(Alert) // we need to zero the memory for Alert or it'll contain a borked proto unmarshal
		log.Printf("JSON is [%s]", string(payload))
		err := json.Unmarshal(payload,alert)
		if err != nil {
			// failed as JSON too, error
			return alert,fmt.Errorf("Couldn't understand packet.")
		} else {
			// JSON packet
			log.Printf("Decoded alert as JSON: %v", alert)
			return alert,nil
		}
	} else {
		// ok, it's proto, return it
		return alert,nil
	}
}

func convertStreaming(baseTopic string, inc <-chan mqtt.Message, out chan<- *AlertUpdate) {
	for {
		m := <-inc
		alert,err := unmarshalAlert(m.Payload())
		if err != nil {
			log.Printf("Skipping packet that failed to unmarshal")
		} else {
			source, _, _ := ParseAlertTopic(baseTopic, m.Topic())
			up := CreateUpdate(source, false, alert)
			log.Printf("Got %v", alert)
			out <- up
		}
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
		} else {
			log.Printf("Sent %s@%s/%s to Mauve.", *up.Alert[0].Id, *up.Source, *up.Alert[0].Subject)
		}
	}
}

func dialMQTT(broker string, baseTopic string) (*mqtt.MqttClient, chan mqtt.Message) {
	incomingMessages := make(chan mqtt.Message)
	hostname,_ :+ os.Hostname()
	clientId := fmt.Sprintf("govealert-mqtt-receiver-%s",hostname)
	mqttOpts := mqtt.NewClientOptions().AddBroker(broker).SetClientId(clientId).SetCleanSession(false).SetOnConnectionLost(mqttDisconnect)
	filter, _ := mqtt.NewTopicFilter(fmt.Sprintf("%s/+/+/+", baseTopic), byte(1))
	mqttOpts.SetDefaultPublishHandler(func(client *mqtt.MqttClient, msg mqtt.Message) {
		log.Printf("Packet on %s", msg.Topic())
		incomingMessages <- msg
	})
	willPacket := mqttStatusPacket(false)
	mqttOpts.SetBinaryWill(mqttHeartbeatTopic(baseTopic), willPacket.Payload(), mqtt.QOS_ONE, true)
	client := mqtt.NewClient(mqttOpts)
	if _, err := client.Start(); err != nil {
		log.Fatalf("Failed to connect to MQTT Broker: %s - %s", broker, err)
	} else {
		log.Printf("Connected to Broker")
		if _, err := client.StartSubscription(nil, filter); err != nil {
			log.Printf("Failed to subscribe: %s", err)
		}
	}
	return client,incomingMessages
}

func mqttHeartbeatTopic(baseTopic string) string {
	hostname, _ := os.Hostname()
    return fmt.Sprintf("%s/$heartbeat/%s", baseTopic, hostname)
}

func mqttStatusPacket(running bool) *mqtt.Message {
	hostname, _ := os.Hostname()
	now := time.Now().Unix()
	status := map[string]string{
		"hostname": hostname,
		"now":      strconv.FormatInt(now, 10),
		"running":  strconv.FormatBool(running),
	}
	json, _ := json.Marshal(status)
	m := mqtt.NewMessage(json)
	m.SetRetainedFlag(true)
	return m
}

func mqttHeartbeat(topicBase string, secs time.Duration, client *mqtt.MqttClient) {
	for {
		s := mqttStatusPacket(true)
		publishTopic := mqttHeartbeatTopic(topicBase)
		log.Printf("Publishing heartbeat to %s", publishTopic)
		client.PublishMessage(publishTopic, s)
		time.Sleep(secs)
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
	mqttTopic := flag.String("mqtt-base", "govealert", "Base topic for MQTT transport packets")
	flag.Parse()

	msend := make(chan *AlertUpdate, 50)    // the channel we'll dump AlertUpdate packets destined for Mauve into
	go dialMauve(false, *mauvealert, msend) // this goroutine will send any packets on the msend channel into mauve

	mq,incomingAlerts := dialMQTT(*mqttBroker, *mqttTopic)
	go mqttHeartbeat(*mqttTopic, time.Duration(60)*time.Second,mq)

	convertedAlerts := make(chan *AlertUpdate)
	go convertStreaming(*mqttTopic, incomingAlerts, convertedAlerts)

	var inc *AlertUpdate

	for {
		inc = <-convertedAlerts
		log.Printf("Passing on alertUpdate as: %v", inc)
		msend <- inc
	}
}
