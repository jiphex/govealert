package mauve

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

type MauveAlertService struct {
	Host string
	Port uint16
}

// Wrap a single Alert in an AlertUpdate message, with the
// appropriate source and replace flags set.
func CreateUpdate(source string, replace bool, alerts ...*Alert) *AlertUpdate {	
	transmissionID := randomTransmissionId()
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
func ParseTimeWithNow(raw string) (time.Duration,error) {
	// This parses 
	if raw == "" {
		return 0,fmt.Errorf("Invalid empty time string")
	} else if raw == "now" {
		return 0,nil
	} else {
		d, err := time.ParseDuration(raw)
		if err != nil {
			// log.Fatalf("Failed to parse raise time: ")
			return 0,err
		} else {
			return d,nil
		}
	}
}

func CreateAlert(id string, raise string, clear string, subject string, summary string, detail string, suppress string) (*Alert,error) {
	var tRaise, tClear, tSuppress uint64
	if raise != "" {
		tDiff,err := ParseTimeWithNow(raise)
		if err != nil {
			return nil,fmt.Errorf("Problem with raise time: %s", err)
		}
		tRaise = uint64(time.Now().Add(tDiff).Unix())
	}
	if clear != "" {
		tDiff,err := ParseTimeWithNow(clear)
		if err != nil {
			return nil,fmt.Errorf("Problem with clear time: %s", err)
		}
		tClear = uint64(time.Now().Add(tDiff).Unix())
	}
	if suppress != "" {	
		tDiff,err := ParseTimeWithNow(suppress)
		if err != nil {
			return nil,fmt.Errorf("Problem with suppress time: %s", err)
		}
		tSuppress = uint64(time.Now().Add(tDiff).Unix())
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
	return &alert,nil
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
