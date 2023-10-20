package syslogsidecar

import (
	"fmt"
	"strconv"
	"time"

	"github.com/g41797/go-syslog/format"
	"github.com/g41797/sputnik"
)

type syslogmsgparts struct {
	parts
}

func (mp *syslogmsgparts) Extract(ef func(name, value string) error) error {

	count, _ := mp.runeAt(0)

	switch int(count) {
	case BadMessageParts:
		return mp.extractBadMessage(ef)
	case RFC5424Parts:
		return mp.extractRFCMessage(rfc5424names[:], ef)
	case RFC3164Parts:
		return mp.extractRFCMessage(rfc3164names[:], ef)
	}

	return fmt.Errorf("Wrong packed syslog message")
}

func (mp *syslogmsgparts) extractBadMessage(extr func(name, value string) error) error {
	mlen, _ := mp.runeAt(1)
	mp.skip(2)

	value, err := mp.part(int(mlen))

	if err != nil {
		return err
	}

	return extr(FormerMessage, value)
}

// func (mp *syslogmsgparts) extractRFC5424Message(extr func(name, value string) error) error {
// 	return nil
// }

// func (mp *syslogmsgparts) extractRFC3164Message(extr func(name, value string) error) error {
// 	return nil
// }

func (mp *syslogmsgparts) extractRFCMessage(names []string, extr func(name, value string) error) error {
	return nil
}

func (mp *syslogmsgparts) fillMsg(logParts format.LogParts, msgLen int64, err error) bool {

	if logParts == nil {
		return false
	}

	if len(logParts) == 0 {
		return false
	}

	if err != nil {
		mp.fillFormerMessage(logParts)
		return true
	}

	if _, exists := logParts[RFC5424OnlyKey]; exists {
		mp.fillRFC5424(logParts)
		return true
	}

	if _, exists := logParts[RFC3164OnlyKey]; exists {
		mp.fillRFC3164(logParts)
		return true
	}
	mp.fillFormerMessage(logParts)
	return true
}

func (mp *syslogmsgparts) set() {
	if len(mp.data) == 0 {
		mp.data = make([]rune, 128)
	}
}

func (mp *syslogmsgparts) fillFormerMessage(logParts format.LogParts) {
	mp.set()
	mp.setRuneAt(0, 1)
	mp.skip(2)
	mp.setRuneAt(1, rune(mp.appendText(logParts[FormerMessage].(string))))
}

func (mp *syslogmsgparts) fillRFC5424(logParts format.LogParts) {
	mp.set()

	count := 1 + len(rfc5424names)
	mp.setRuneAt(0, rune(count))
	mp.skip(count + 1)

	mp.setRuneAt(1, rune(mp.appendText(RFC5424)))

	for i, name := range rfc5424names {
		v, exists := logParts[name]

		if !exists {
			mp.setRuneAt(i+2, 0)
			continue
		}

		mp.setRuneAt(i+2, rune(mp.appendText(toString(v, rfc5424props[name]))))
	}
}

func (mp *syslogmsgparts) fillRFC3164(logParts format.LogParts) {
	mp.set()

	count := 1 + len(rfc3164names)
	mp.setRuneAt(0, rune(count))
	mp.skip(count + 1)

	mp.setRuneAt(1, rune(mp.appendText(RFC3164)))

	for i, name := range rfc3164names {
		v, exists := logParts[name]

		if !exists {
			mp.setRuneAt(i+2, 0)
			continue
		}

		mp.setRuneAt(i+2, rune(mp.appendText(toString(v, rfc3164props[name]))))
	}

}

const (
	RFC3164OnlyKey  = "tag"
	RFC5424OnlyKey  = "structured_data"
	RFCFormatKey    = "rfc"
	RFC3164         = "RFC3164"
	RFC5424         = "RFC5424"
	SeverityKey     = "severity"
	ParserError     = "parsererror"
	FormerMessage   = "data"
	BrokenParts     = 2
	BadMessageParts = 1
	RFC5424Parts    = 12 // RFC + 11
	RFC3164Parts    = 8  // RFC + 7
)

//
// https://blog.datalust.co/seq-input-syslog/
//

// ------------------------------------
// priority = (facility * 8) + severity
// ------------------------------------

// RFC3164 parameters with type
// https://documentation.solarwinds.com/en/success_center/kss/content/kss_adminguide_syslog_protocol.htm
func RFC3164Props() map[string]string {
	return map[string]string{
		"priority":     "int",
		"facility":     "int",
		SeverityKey:    "int",
		"timestamp":    "time.Time",
		"hostname":     "string",
		RFC3164OnlyKey: "string",
		"content":      "string",
	}
}

var rfc3164props = RFC3164Props()

var rfc3164names = [7]string{
	"priority", "facility", SeverityKey, "timestamp", "hostname", RFC3164OnlyKey, "content"}

// RFC5424 parameters with type
// https://hackmd.io/@njjack/syslogformat
func RFC5424Props() map[string]string {
	return map[string]string{
		"priority":     "int",
		"facility":     "int",
		SeverityKey:    "int",
		"version":      "int",
		"timestamp":    "time.Time",
		"hostname":     "string",
		"app_name":     "string",
		"proc_id":      "string",
		"msg_id":       "string",
		RFC5424OnlyKey: "string",
		"message":      "string",
	}
}

var rfc5424props = RFC5424Props()

var rfc5424names = [11]string{
	"priority", "facility", SeverityKey, "version", "timestamp", "hostname",
	"app_name", "proc_id", "msg_id", RFC5424OnlyKey, "message"}

func toMsg(logParts format.LogParts, msgLen int64, err error) sputnik.Msg {
	if logParts == nil {
		return nil
	}

	if len(logParts) == 0 {
		return nil
	}

	if err != nil {
		return msgFromFormerMsg(logParts)
	}

	if _, exists := logParts[RFC5424OnlyKey]; exists {
		return toRFC5424(logParts)
	}

	if _, exists := logParts[RFC3164OnlyKey]; exists {
		return toRFC3164(logParts)
	}

	return msgFromFormerMsg(logParts)

}

func msgFromFormerMsg(logParts format.LogParts) sputnik.Msg {
	msg := make(sputnik.Msg)
	formerMsg := logParts[FormerMessage].(string)
	msg[FormerMessage] = formerMsg
	return msg
}

// Convert syslog RFC5424 values to strings
func toRFC5424(logParts format.LogParts) sputnik.Msg {
	msg := make(sputnik.Msg)
	msg[RFCFormatKey] = RFC5424

	for _, name := range rfc5424names {
		v, exists := logParts[name]
		if !exists {
			msg[name] = ""
			continue
		}
		msg[name] = toString(v, rfc5424props[name])
	}

	return msg
}

// Convert syslog RFC3164 values to strings
func toRFC3164(logParts format.LogParts) sputnik.Msg {
	msg := make(sputnik.Msg)
	msg[RFCFormatKey] = RFC3164

	for _, name := range rfc3164names {
		v, exists := logParts[name]
		if !exists {
			msg[name] = ""
			continue
		}
		msg[name] = toString(v, rfc3164props[name])
	}

	return msg
}

func toString(val any, typ string) string {
	result := ""

	if val == nil {
		return result
	}

	switch typ {
	case "string":
		result, _ = val.(string)
		return result
	case "int":
		intval, _ := val.(int)
		result = strconv.Itoa(intval)
		return result
	case "time.Time":
		tval, _ := val.(time.Time)
		result = tval.UTC().String()
		return result
	}

	return result
}
