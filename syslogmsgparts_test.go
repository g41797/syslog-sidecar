package syslogsidecar

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/g41797/go-syslog/format"
)

func Test_PackUnpackBadlyFormatted(t *testing.T) {

	in := map[string]string{
		Formermessage: "Formermessage",
	}

	logparts, err := toLogParts(in, formerMessage[:])
	if err != nil {
		t.Fatalf("toLogParts error %v", err)
	}

	msgparts := new(syslogmsgparts)

	err = msgparts.pack(logparts)
	if err != nil {
		t.Errorf("pack error %v", err)
	}

	hlp := NewUnpackHelper()

	err = msgparts.Unpack(hlp.Put)
	if err != nil {
		t.Errorf("Unpack error %v", err)
	}

	if !reflect.DeepEqual(in, hlp.LogParts) {
		t.Errorf("Expected %v Actual %v", in, hlp.LogParts)
	}
}

func Test_PackUnpackRFC3164Msg(t *testing.T) {
	testPackUnpackRFCMsg(makeRFC3164Msg(), rfc3164parts[:], t)
}

func Test_PackUnpackRFC5424Msg(t *testing.T) {
	testPackUnpackRFCMsg(makeRFC5424Msg(), rfc5424parts[:], t)
}

func testPackUnpackRFCMsg(in map[string]string, descr []partType, t *testing.T) {

	logparts, err := toLogParts(in, descr)
	if err != nil {
		t.Fatalf("toLogParts error %v", err)
	}

	msgparts := new(syslogmsgparts)

	err = msgparts.pack(logparts)
	if err != nil {
		t.Errorf("pack error %v", err)
	}

	hlp := NewUnpackHelper()

	err = msgparts.Unpack(hlp.Put)
	if err != nil {
		t.Errorf("Unpack error %v", err)
	}

	if !reflect.DeepEqual(in, hlp.LogParts) {
		t.Errorf("Expected %v Actual %v", in, hlp.LogParts)
	}
}

func makeRFC3164Msg() map[string]string {

	msg := map[string]string{
		rfcFormatKey:   rfc3164,
		"priority":     "1",
		"facility":     "2",
		severityKey:    "3",
		"timestamp":    time.Now().Format(time.RFC3339),
		"hostname":     "4",
		rfc3164OnlyKey: "5",
		"content":      "content",
	}

	return msg
}

func makeRFC5424Msg() map[string]string {

	msg := map[string]string{
		rfcFormatKey:   rfc5424,
		"priority":     "1",
		"facility":     "2",
		severityKey:    "3",
		"version":      "4",
		"timestamp":    time.Now().Format(time.RFC3339),
		"hostname":     "5",
		"app_name":     "6",
		"proc_id":      "7",
		"msg_id":       "8",
		rfc5424OnlyKey: "9",
		"message":      "message",
	}

	return msg
}

func toLogParts(in map[string]string, parts []partType) (format.LogParts, error) {
	if len(in) == 0 {
		return nil, fmt.Errorf("empty in")
	}

	logParts := make(format.LogParts)

	for _, part := range parts {
		val, exists := in[part.name]
		if !exists {
			return nil, fmt.Errorf("%s does not exist", part.name)
		}
		logParts[part.name] = toValue(val, part.kind)
	}

	return logParts, nil
}

func toValue(str string, typ string) any {

	switch typ {
	case "string":
		return str
	case "int":
		result, _ := strconv.Atoi(str)
		return result
	case "time.Time":
		result, _ := time.Parse(time.RFC3339, str)
		return result
	}

	return nil
}
