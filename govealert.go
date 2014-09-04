package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"time"

	"code.google.com/p/goprotobuf/proto"
	"repos.bytemark.co.uk/govealert/mauve"
	"repos.bytemark.co.uk/mauvesend.go/golang-timeinput-dev"
)

func CreateAlert(id *string, raise *string, clear *string, subject *string, summary *string, detail *string, suppress *string) *mauve.Alert {
	alert := &mauve.Alert{
		Id: id,
	}

	if *raise != "" {
		rt, err := timeinput.RelativeUnixTime(*raise)
		if err != nil {
			log.Fatalf("Failed to parse raise time.")
		} else {
			alert.RaiseTime = &rt
		}
	}

	if *clear != "" {
		ct, err := timeinput.RelativeUnixTime(*clear)
		if err != nil {
			log.Fatalf("Failed to parse clear time.")
		} else {
			alert.ClearTime = &ct
		}
	}

	if *suppress != "" {
		st, err := timeinput.RelativeUnixTime(*suppress)
		if err != nil {
			log.Fatalf("Failed to parse suppress time.")
		} else {
			alert.SuppressUntil = &st
		}
	}

	if *subject != "" {
		alert.Subject = subject
	} else {
		hn, _ := os.Hostname()
		alert.Subject = &hn
	}
	if *summary != "" {
		alert.Summary = summary
	}
	if *detail != "" {
		alert.Detail = detail
	}
	return alert
}

func randomTransmissionId() uint64 {
	// Just make a random number, used for the AlertUpdate creation.
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return uint64(r.Int63())
}

func CreateUpdate(source *string, replace *bool, alert *mauve.Alert) *mauve.AlertUpdate {
	// Wrap a single Alert in an AlertUpdate message, with the
	// appropriate source and replace flags set.
	transmissionID := randomTransmissionId()
	alerts := make([]*mauve.Alert, 0)
	alerts = append(alerts, alert)
	now := uint64(time.Now().Unix())
	update := &mauve.AlertUpdate{
		Source:           source,
		Replace:          replace,
		Alert:            alerts,
		TransmissionId:   &transmissionID,
		TransmissionTime: &now,
	}
	return update
}

func DialMauve(source *string, replace *bool, host *string, queue <-chan *mauve.Alert, receipt chan<- uint64) {
	// This connects to Mauve over UDP and then waits on it's channel,
	// any Alert that gets written to the channel will get wrapped in
	// an AlertUpdate and then sent to the Mauve server
	addr, err := net.ResolveUDPAddr("udp", *host)
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
		au := <-queue
		up := CreateUpdate(source, replace, au)
		mu, _ := proto.Marshal(up)
		//log.Printf("Sent: %s", up.String())
		conn.Write(mu)
		//log.Printf("sent %d bytes to %s", len(mu), *host)
		receipt <- *up.TransmissionId
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
	mauvealert := flag.String("mauve", "alert.bytemark.co.uk:32741", "Mauve Server to dial")
	suppress := flag.String("suppress", "", "Suppress alert for the specified time")
	heartbeat := flag.Bool("heartbeat", false, "Don't do normal operation, just send a 10 minute heartbeat")
	cancel := flag.Bool("cancel", false, "When specified with -heartbeat, cancels the heartbeat (via suppress+raise, clear)")
	flag.Parse()
	if len(*clear) > 0 {
		*raise = ""
	}
	msend := make(chan *mauve.Alert)
	receipt := make(chan uint64)
	go DialMauve(source, replace, mauvealert, msend, receipt)
	if *heartbeat {
		hbsumm := fmt.Sprintf("heartbeat failed for %s", hostname)
		hbdetail := fmt.Sprintf("The govealert heartbeat wasn't sent for the host %s.", hostname)
		hbid := "heartbeat"
		hbraise := "+10m"
		hbclear := ""
		if *cancel {
			// Cancel a heartbeat alert by sending: suppressed raise, clear (experimental)
			//log.Printf("Cancelling alert heartbeat")
			supraise := "now"
			suptime := "+5m"
			sup := CreateAlert(&hbid, &supraise, &hbclear, &hostname, &hbsumm,&hbdetail, &suptime)
			msend <- sup
			<-receipt
			clr := CreateAlert(&hbid, &hbclear, &supraise, &hostname, &hbsumm, &hbdetail, &hbclear)
			msend <- clr
			<-receipt
		} else {
			// 	Send a hearbeat alert
			al := CreateAlert(&hbid, &hbraise, &hbclear, &hostname, &hbsumm, &hbdetail, &hbclear)
			msend <- al
			<-receipt
		}
	} else {
		custom := CreateAlert(id, raise, clear, subject, summary, detail, suppress)
		msend <- custom
		<-receipt
	}
}
