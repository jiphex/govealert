package mauve

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
	"net"
)

type MauveAlertService struct {
	Host string
	Port uint16
}

func LookupMauvesForDomain(domain string) ([]*MauveAlertService, error) {
	cname,addrs,err := net.LookupSRV("mauve", "udp", domain)
	if err != nil {
		return nil,fmt.Errorf("Resolution error: %s", err)
	}
	if len(addrs) > 0 {
		ret := make([]*MauveAlertService,len(addrs))
		for i,srv := range addrs {
			ret[i] = &MauveAlertService{
				Host: srv.Target[0:len(srv.Target)-1],
				Port: srv.Port,
			}
		}
		return ret,nil
	} else {
		return nil,fmt.Errorf("Failed to find any Mauve records at %s", cname)
	}
}

// Wrap a single Alert in an AlertUpdate message, with the
// appropriate source and replace flags set.
func CreateUpdate(source string, replace bool, alert *Alert) *AlertUpdate {	
	transmissionID := randomTransmissionId()
	alerts := []*Alert{alert}
	now := uint64(time.Now().Unix())
	update := &AlertUpdate{
		Source:           &source,
		Replace:          &replace,
		Alert:            alerts,
		TransmissionId:   &transmissionID,
		TransmissionTime: &now,
	}
	return update
}

// Just make a random number, used for the AlertUpdate creation.
func randomTransmissionId() uint64 {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return uint64(r.Int63())
}

// This is identical to time.ParseDuration(string), however it can also take 
// the single string specifier "now" which should always return 0
func ParseTimeWithNow(raw string) time.Duration {
	// This parses 
	if raw == "" {
		panic(fmt.Sprintf("Empty duration: [%s]", raw))
	} else if raw == "now" {
		return 0
	} else {
		d, err := time.ParseDuration(raw)
		if err != nil {
			panic(err)
		} else {
			return d
		}
	}
	// won't get here
	return 0
}

func CreateAlert(id string, raise string, clear string, subject string, summary string, detail string, suppress string) *Alert {
	var tRaise, tClear, tSuppress uint64
	if raise != "" {
		tRaise = uint64(time.Now().Add(ParseTimeWithNow(raise)).Unix())
	}
	if clear != "" {
		tClear = uint64(time.Now().Add(ParseTimeWithNow(clear)).Unix())
	}
	if suppress != "" {
		tSuppress = uint64(time.Now().Add(ParseTimeWithNow(suppress)).Unix())
	}
	alert := Alert{
		Id:            &id,
		RaiseTime:     &tRaise,
		ClearTime:     &tClear,
		SuppressUntil: &tSuppress,
	}

	if subject != "" {
		alert.Subject = &subject
	} else {
		hn, _ := os.Hostname()
		alert.Subject = &hn
	}
	if summary != "" {
		alert.Summary = &summary
	}
	if detail != "" {
		alert.Detail = &detail
	}
	return &alert
}

func AlertTopic(al *Alert, source string) string {
	esource := strings.Replace(source, "/", "_", -1)
	esubj := strings.Replace(*al.Subject, "/", "_", -1)
	eid := strings.Replace(*al.Id, "/", "_", -1)
	return fmt.Sprintf("%s/%s/%s", esource, esubj, eid)
}

func ParseAlertTopic(baseTopic string, topic string) (source string, subject string, id string) {
	lBase := len(strings.Split(baseTopic, "/")) // todo: deal with leading/trailing slashes
	parts := strings.SplitN(topic, "/", lBase+3)
	return parts[1], parts[2], parts[3]
}
