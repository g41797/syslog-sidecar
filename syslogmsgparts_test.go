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

	err = msgparts.pack(logparts, fmt.Errorf("bad formatted message"))
	if err != nil {
		t.Errorf("pack error %v", err)
	}

	hlp := newpuhelper()

	err = msgparts.Unpack(hlp.put)
	if err != nil {
		t.Errorf("Unpack error %v", err)
	}

	if !reflect.DeepEqual(in, hlp.slmparts) {
		t.Errorf("Expected %v Actual %v", in, hlp.slmparts)
	}
}

type puhelper struct {
	slmparts map[string]string
}

func newpuhelper() puhelper {
	var result puhelper
	result.slmparts = make(map[string]string)
	return result
}

func (hlp *puhelper) put(name, value string) error {
	if hlp.slmparts == nil {
		return fmt.Errorf("nil slmparts")
	}

	if _, present := hlp.slmparts[name]; present {
		return fmt.Errorf("%s already exists", name)
	}

	hlp.slmparts[name] = value
	return nil
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
