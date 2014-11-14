package mauve

import (
	"testing"
	"time"
)

func TestCreateAlert(t *testing.T) {
	//func CreateAlert(id string, raise string, clear string, subject string, summary string, detail string, suppress string) *Alert {
	testId := "exampleTesting"
	testRaiseTime := "now"
	testClearTime := ""
	subject := "subject.example.com"
	summary := "This is a test of the CreateAlert function"
	detail := "This is a test of the CreateAlert function and has some text in the detail"
	suppress := ""
	a,err := CreateAlert(testId,testRaiseTime,testClearTime,subject,summary,detail,suppress)
	if err != nil {
		t.Fatal("Alert creation failed: %s", err)
	}
	if *a.Id != testId {
		t.Errorf("Created alert ID didn't match")
	}
	//todo: incomplete test
}

func TestCreateUpdate(t *testing.T) {
	sourceTest := "test.example.com"
	replace := false
	fakeAlert,err := CreateAlert("id","now","","","","","")
	if err != nil {
		t.Fatal("Alert created with error: %s", err)
	}
	u := CreateUpdate(sourceTest, replace, fakeAlert)
	if *u.Source != sourceTest {
		t.Errorf("Mismatched source after createUpdate")
	}
	// todo: incomplete test
}

func TestParseTimeWithNow(t *testing.T) {
	testCases := map[string]time.Duration {
		"now": 0,
		"10s": time.Duration(10)*time.Second,
		"3m": time.Duration(3)*time.Minute,
		"6h": time.Duration(6)*time.Hour,
	}
	for tVal,expected := range testCases {
		xres,err := ParseTimeWithNow(tVal)
		if err != nil {
			t.Fatal("Failed to parse time")
		}
		if xres != time.Duration(expected) {
			t.Errorf("%d does not match %v for test [%s]", xres, expected, tVal)
		}
	}
}

type ttAlertSSID struct {
	Source string
	Subject string
	Id string
}

func TestAlertTopic(t *testing.T) {
	testCases := map[ttAlertSSID]string {
		ttAlertSSID{"tSource","tSubject","tId"}: "tSource/tSubject/tId", // normal
		ttAlertSSID{"asdax/foo", "bar", "baz"}: "asdax_foo/bar/baz", // contains a slash in the source that should be escaped
		ttAlertSSID{"asda", "barx/foo", "baz"}: "asda/barx_foo/baz", // contains a slash in the subject that should be escaped
		ttAlertSSID{"asda", "bar", "bazx/foo"}: "asda/bar/bazx_foo", // contains a slash in the id that should be escaped
	}
	for tass, expt := range testCases {
		tAlert,err := CreateAlert(tass.Id, "", "", tass.Subject, "", "", "")
		if err != nil {
			t.Fatal("Failed to create alert: %s", err)
		}
		xret := AlertTopic(tAlert, tass.Source)
		if xret != expt {
			t.Errorf("Failed to get proper alert topic t, %s is not %s", xret, expt)
		}
	}
}

func TestParseAlertTopic(t *testing.T) {
	testBase := "testttestest"
	testCases := map[string][]string {
		"testttestest/foo/bar/baz": []string{"foo", "bar", "baz"}, // e.g testttestest/foo/bar/baz => t,f,ba,bz
		"testttestest/foooooooooo": nil, // value of 'nil' means we're expecting this to error
		"testttestest/foo/bar/baz/boo": []string{"foo", "bar", "baz/boo"},
	}
	for testTopic,expected := range testCases {
		rSource,rSubject,rId,err := ParseAlertTopic(testBase,testTopic)
		if err == nil {
			if !SlicesEqual([]string{rSource,rSubject,rId},expected) {
				t.Errorf("Mismatch with ParseAlertTopic from %s/%s (is %s, %s, %s)", testBase,testTopic,rSource,rSubject,rId)
			}
		} else {
			if expected != nil { // if expected is nil then we're expecting an error
				t.Fatal("Error returned from %s", testTopic)
			}
		}
	}
}

func SlicesEqual(sliceA []string, sliceB []string) bool {
	if len(sliceA) != len(sliceB) {
		return false
	}
	for i,x := range sliceA {
		if sliceB[i] != x {
			return false
		}
	}
	return true
}