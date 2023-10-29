package syslogsidecar

import (
	"fmt"

	"github.com/g41797/sputnik"
)

// Creates map[string]string from syslog parts are stored within message
func UnpackToMap(msg sputnik.Msg) (map[string]string, error) {
	uh := NewUnpackHelper()
	err := Unpack(msg, uh.Put)
	if err != nil {
		return nil, err
	}
	return uh.LogParts, nil
}

// For every part of syslog message(stored within msg) runs supplied callback
// See README for the list of partnames
func Unpack(msg sputnik.Msg, f func(partname string, val string) error) error {
	if msg == nil {
		return fmt.Errorf("nil msg")
	}

	slm, exists := msg[syslogmessage]
	if !exists {
		return fmt.Errorf("empty msg")
	}

	syslogmsgparts, ok := slm.(*syslogmsgparts)
	if !ok {
		return fmt.Errorf("wrong msg")
	}

	return syslogmsgparts.Unpack(f)
}

func Pack(msg sputnik.Msg, parts map[string]string) error {
	if msg == nil {
		return fmt.Errorf("nil msg")
	}

	count := len(parts)

	if count == 0 {
		return fmt.Errorf("empty parts")
	}

	syslogmsgparts, exists := msg[syslogmessage].(*syslogmsgparts)
	if !exists {
		syslogmsgparts = newsyslogmsgparts()
		msg[syslogmessage] = syslogmsgparts
	}

	var partsDescr []partType

	switch count {
	case badMessageParts:
		partsDescr = formerMessage[:]
	case rfc5424Parts:
		partsDescr = rfc5424parts[:]
	case rfc3164Parts:
		partsDescr = rfc3164parts[:]
	default:
		return fmt.Errorf("wrong parts")
	}

	return pack(msg, parts, syslogmsgparts, partsDescr)
}

func pack(msg sputnik.Msg, parts map[string]string, syslogmsgparts *syslogmsgparts, expected []partType) error {

	count := len(expected)
	syslogmsgparts.setRuneAt(0, rune(count))
	syslogmsgparts.skip(count + 1)

	for i, part := range expected {
		val, exists := parts[part.name]

		if !exists {
			return fmt.Errorf("%s does not exist", part.name)
		}

		syslogmsgparts.setRuneAt(i+1, rune(syslogmsgparts.appendText(val)))
	}
	return nil
}

type UnpackHelper struct {
	LogParts map[string]string
}

func NewUnpackHelper() UnpackHelper {
	var result UnpackHelper
	result.LogParts = make(map[string]string)
	return result
}

func (hlp *UnpackHelper) Put(name, value string) error {
	if hlp.LogParts == nil {
		return fmt.Errorf("nil LogParts")
	}

	if _, present := hlp.LogParts[name]; present {
		return fmt.Errorf("%s already exists", name)
	}

	hlp.LogParts[name] = value
	return nil
}
