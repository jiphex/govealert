package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"encoding/json"

	"code.google.com/p/goprotobuf/proto"
	. "repos.bytemark.co.uk/govealert/mauve"
)

func DialMQTT(source string, broker string, topicBase string, queue <-chan *Alert, receipt chan<- uint64) {
	for {
		al := <-queue
		json,err := json.Marshal(al)
		if err != nil {
			log.Fatalf("JSON Marshalling fail: %v", err)
		} else {
			log.Printf("Alert: %s", json)
		}
		// send the packet
		log.Printf("Sending MQTT transport packet: %s", al)
	}
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
		if(err != nil) {
			log.Fatalf("Failed to marshal an alertUpdate: %s", err)
		}
		//log.Printf("Sent: %s", up.String())
		if _,err := conn.Write(mu); err != nil {
			log.Fatalf("Failed to send message: %s", err)
		} else {
			receipt <- *up.TransmissionId
		}
	}
}

func main() {
	hostname, _ := os.Hostname()
	id := flag.String("id", "govealert", "Alert ID to send")
	subject := flag.String("subject", hostname, "What the alert is about")
	summary := flag.String("summary", "", "Short text desription of the alert")
	detail := flag.String("detail", "", "Longer textual description of the alert")
	source := flag.String("source", hostname, "The thing that generated the alert")
	raise := flag.String("raise", "now", "Time to raise the alert")
	clear := flag.String("clear", "", "Time to clear the alert")
	replace := flag.Bool("replace", false, "Replace all alerts for this subject")
	mauvealert := flag.String("mauve", "alert.bytemark.co.uk:32741", "Mauve (or MQTT) Server to dial")
	suppress := flag.String("suppress", "", "Suppress alert for the specified time")
	heartbeat := flag.Bool("heartbeat", false, "Don't do normal operation, just send a 10 minute heartbeat")
	cancel := flag.Bool("cancel", false, "When specified with -heartbeat, cancels the heartbeat (via suppress+raise, clear)")
	mqtt := flag.Bool("mqtt", false, "Whether to use the (experimental) MQTT transport instead of protobuf")
	mqttBroker := flag.String("mqtt-broker", "tcp://localhost:1883", "The MQTT Broker to connect to")
	mqttTopic := flag.String("mqtt-base", "/govealert", "Base topic for MQTT transport packets")
	flag.Parse()
	if len(*clear) > 0 {
		*raise = ""
	}
	msend := make(chan *Alert, 5)
	// This is just a channel we wait on to make sure we only send one alert at once
	receipt := make(chan uint64)
	if *mqtt {
		go DialMQTT(*source, *mqttBroker, *mqttTopic, msend, receipt)
	} else {
		go DialMauve(*source, *replace, *mauvealert, msend, receipt)
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
			sup := CreateAlert(hbid, supraise, hbclear, hostname, hbsumm,hbdetail, suptime)
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
