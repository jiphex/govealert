package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"code.google.com/p/go.net/publicsuffix"
	
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

func DialMQTT(source string, broker string, topicBase string, queue <-chan *Alert, receipt chan<- uint64, marshalAs string) {
	mqttOpts := mqtt.NewClientOptions().AddBroker(broker).SetClientId("govealert").SetCleanSession(true).SetOnConnectionLost(mqttDisconnect)
	client := mqtt.NewClient(mqttOpts)
	if _, err := client.Start(); err != nil {
		log.Fatalf("Failed to connect to MQTT Broker: %s - %s", broker, err)
	} else {
		log.Printf("Connected to Broker")
	}
	for {
		al := <-queue
		var pkt []byte
		var err error
		if marshalAs == "json" {
			pkt, err = json.Marshal(al)
		} else {
			pkt, err = proto.Marshal(al)
		}
		if err != nil {
			log.Fatalf("Marshalling fail: %v", err)
		}
		// send the packet
		log.Printf("Sending MQTT transport packet: %s", al)
		fullTopic := fmt.Sprintf("%s/%s", topicBase, alertTopic(al, source))
		mqttreceipt := client.Publish(mqtt.QOS_ONE, fullTopic, pkt)
		<-mqttreceipt
		receipt <- 0
		log.Printf("Sent MQTT transport packet: %s", al)
	}
}

func DialMauve(source string, replace bool, host *MauveAlertService, queue <-chan *Alert, receipt chan<- uint64) {
	// This connects to Mauve over UDP and then waits on it's channel,
	// any Alert that gets written to the channel will get wrapped in
	// an AlertUpdate and then sent to the Mauve server
	mauveIP,err := net.ResolveIPAddr("ip",host.Host)
	addr := &net.UDPAddr{mauveIP.IP,int(host.Port),mauveIP.Zone}
	if err != nil {
		log.Fatalf("Cannot resolve mauvealert server: %s", addr)
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Fatalf("Failed to connect to mauve: %s", addr)
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
		if bytes, err := conn.Write(mu); err != nil {
			log.Fatalf("Failed to send message: %s", err)
		} else {
			log.Printf("Sent %d bytes to %s:%d", bytes, host.Host, host.Port)
			receipt <- *up.TransmissionId
		}
	}
}

func main() {
	hostname, _ := os.Hostname()
	// we need to do some wrangling to get the default domain
	psname,err := publicsuffix.EffectiveTLDPlusOne(hostname)
	if err != nil {
		psname = hostname // shrug
	} 
	id := flag.String("id", "govealert", "Alert ID to send")
	subject := flag.String("subject", hostname, "What the alert is about")
	summary := flag.String("summary", "", "Short text desription of the alert")
	detail := flag.String("detail", "", "Longer textual description of the alert")
	source := flag.String("source", hostname, "The thing that generated the alert")
	raise := flag.String("raise", "now", "Time to raise the alert")
	clear := flag.String("clear", "", "Time to clear the alert")
	replace := flag.Bool("replace", false, "Replace all alerts for this subject")
	mauvealert := flag.String("mauve", psname, "Mauve server to dial (will lookup _mauve._udp SRV record of this domain)")
	suppress := flag.String("suppress", "", "Suppress alert for the specified time")
	heartbeat := flag.Bool("heartbeat", false, "Don't do normal operation, just send a 10 minute heartbeat")
	cancel := flag.Bool("cancel", false, "When specified with -heartbeat, cancels the heartbeat (via suppress+raise, clear)")
	transport := flag.String("transport", "protobuf", "Which transport to use, currently one of: protobuf, mqtt")
	mqttBroker := flag.String("mqtt-broker", "tcp://localhost:1883", "The MQTT Broker to connect to")
	mqttTopic := flag.String("mqtt-base", "govealert", "Base topic for MQTT transport packets")
	mqttPublishAs := flag.String("mqtt-marshal", "protobuf", "Marshalling to use for MQTT packets, one of: protobuf, json")
	flag.Parse()
	if len(*clear) > 0 && *raise == "now" {
		*raise = "" // This is supposed to stop the unstated "raise now" if a clear is passed with no raise argument
	}
	msend := make(chan *Alert, 5)
	// This is just a channel we wait on to make sure we only send one alert at once
	receipt := make(chan uint64)
	if *transport == "mqtt" {
		go DialMQTT(*source, *mqttBroker, *mqttTopic, msend, receipt, *mqttPublishAs)
	} else if *transport == "protobuf" {
		mauveHosts,err := LookupMauvesForDomain(*mauvealert)
		// todo: we only look up the first mauve server sadface
		if err != nil {
			log.Fatalf("Mauve Problem: %s", err)
		}
		go DialMauve(*source, *replace, mauveHosts[0], msend, receipt)
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
