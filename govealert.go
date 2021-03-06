package main

import (
	"flag"
	"fmt"
	"log"	
	"os"
	"code.google.com/p/go.net/publicsuffix"
	"github.com/jiphex/govealert/mauve"
)

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
	mode := flag.String("mode", "single", "Sending mode, one of: single, heartbeat")	
	cancel := flag.Bool("cancel", false, "In 'heartbeat' mode, cancels the heartbeat (via suppress+raise, clear)")
	transport := flag.String("transport", "protobuf", "Which transport to use, currently one of: protobuf, mqtt")
	mqttBroker := flag.String("mqttBroker", "tcp://localhost:1883", "The MQTT Broker to connect to")
	mqttTopic := flag.String("mqttBase", "govealert", "Base topic for MQTT transport packets")
	flag.Parse()
	if len(*clear) > 0 && *raise == "now" {
		*raise = "" // This is supposed to stop the unstated "raise now" if a clear is passed with no raise argument
	}
	var client mauve.AlertSender;
	if *transport == "mqtt" {
		client,err = mauve.CreateMQTTClient(*source, *mqttBroker, *mqttTopic)
	} else if *transport == "protobuf" {
		client,err = mauve.CreateProtobufClient(*source,*mauvealert)
	} else {
		log.Fatalf("Unknown alert transport: %s", *transport)
	}
	if err != nil {
		log.Printf("Failed to create %s client: %s", *transport, err)
	}
	if *mode == "heartbeat" {
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
			sup,err := mauve.CreateAlert(hbid, supraise, hbclear, hostname, hbsumm, hbdetail, suptime)
			if err != nil {
				log.Fatalf("Failed to create heartbeat suppression alert: %s", err)
			}
			client.AddBatchedAlert(sup)
			clr,err := mauve.CreateAlert(hbid, hbclear, supraise, hostname, hbsumm, hbdetail, hbclear)
			if err != nil {
				log.Fatalf("Failed to create heartbeat raise+clear alert: %s", err)
			}
			client.AddBatchedAlert(clr)
			client.SendBatchedAlerts(false)
		} else {
			// 	Send a hearbeat alert (clear now, raise in 10 minutes - meant to be called every N where N < 5 minutes)
			al,err := mauve.CreateAlert(hbid, hbraise, hbclear, hostname, hbsumm, hbdetail, hbclear)
			if err != nil {
				log.Fatalf("Failed to send heartbeat: %s", err)
			}
			client.AddBatchedAlert(al)
			client.SendBatchedAlerts(false)
		}
	} else {
		if err != nil {
			log.Fatal(err)
		}
		al,err := mauve.CreateAlert(*id, *raise, *clear, *subject, *summary, *detail, *suppress)
		if err != nil {
			log.Fatalf("Failed to create alert: %s", err)
		}
		client.AddBatchedAlert(al)
		err = client.SendBatchedAlerts(*replace)
		if err != nil {
			log.Fatal(err)
		}
	}
}
