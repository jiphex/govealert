package main

import (
	"flag"
	"log"
	"net"
	"os"
	"time"

	"code.google.com/p/goprotobuf/proto"
	"repos.bytemark.co.uk/govealert/mauve"
	"repos.bytemark.co.uk/mauvesend.go/golang-timeinput-dev"
)

func CreateAlert(id *string, raise *string, clear *string, subject *string, summary *string, detail *string) *mauve.Alert {
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
			alert.RaiseTime = &ct
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

func CreateUpdate(source *string, replace *bool, alert *mauve.Alert) *mauve.AlertUpdate {
	transmissionID := uint64(506157851);
	alerts := []*mauve.Alert{alert}
	now := uint64(time.Now().Unix())
	update := &mauve.AlertUpdate{
		Source:  source,
		Replace: replace,
		Alert:   alerts,
		TransmissionId: &transmissionID,
		TransmissionTime: &now,
	}
	return update
}


func DialMauve(host *string, queue chan *mauve.AlertUpdate) {
	addr, err := net.ResolveUDPAddr("udp", *host)
	if err != nil {
		log.Fatal("Cannot resolve mauvealert server: %s", addr)
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Fatal("Failed to connect to mauve: %s", addr)
	}
	defer conn.Close()
	log.Printf("dialing...")
	for {
		au := <-queue
		mu, _ := proto.Marshal(au)
		log.Printf("Sent: %s", au.String())
		conn.Write(mu)
		log.Printf("sent %d bytes to %s", len(mu), *host)
	}
}

func main() {
	// palert,_ := proto.Marshal(alert)
	hostname, _ := os.Hostname()
	id := flag.String("id", "govesend", "Alert ID to send")
	subject := flag.String("subject", hostname, "What the alert is about")
	summary := flag.String("summary", "", "Short text desription of the alert")
	detail := flag.String("detail", "", "Longer textual description of the alert")
	source := flag.String("source", hostname, "The thing that generated the alert")
	raise := flag.String("raise", "now", "Time to raise the alert")
	clear := flag.String("clear", "", "Time to clear the alert")
	replace := flag.Bool("replace", false, "Replace all alerts for this subject")
	mauvealert := flag.String("mauve", "alert.bytemark.co.uk:32741", "Mauve Server to dial")
	flag.Parse()
	hole := make(chan *mauve.AlertUpdate)
	go DialMauve(mauvealert, hole)
	al := CreateAlert(id, raise, clear, subject, summary, detail)
	up := CreateUpdate(source, replace, al)
	hole <- up
}
