package mauve

import (
	"math/rand"
	"os"
	"time"
	"fmt"
	"text/template"
	"bytes"
)

func CreateUpdate(source string, replace bool, alert *Alert) *AlertUpdate {
	// Wrap a single Alert in an AlertUpdate message, with the
	// appropriate source and replace flags set.
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

func randomTransmissionId() uint64 {
	// Just make a random number, used for the AlertUpdate creation.
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return uint64(r.Int63())
}

func parseTimeWithNow(raw string) time.Duration {
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
		tRaise = uint64(time.Now().Add(parseTimeWithNow(raise)).Unix())
	}
	if clear != "" {
		tClear = uint64(time.Now().Add(parseTimeWithNow(clear)).Unix())
	}
	if suppress != "" {
		tSuppress = uint64(time.Now().Add(parseTimeWithNow(suppress)).Unix())
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
