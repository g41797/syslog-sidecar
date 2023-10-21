package syslogsidecar

import (
	"fmt"
	"strconv"
	"time"

	"github.com/g41797/go-syslog/format"
	"github.com/g41797/sputnik"
)

const (
	Formermessage = "data"

	rfc3164         = "RFC3164"
	rfc5424         = "RFC5424"
	rfcFormatKey    = "rfc"
	rfc3164OnlyKey  = "tag"
	rfc5424OnlyKey  = "structured_data"
	severityKey     = "severity"
	badMessageParts = len(formerMessage)
	rfc5424Parts    = len(rfc5424parts)
	rfc3164Parts    = len(rfc3164parts)
)

type partType struct {
	name string
	kind string
}

// RFC3164 parameters with type
// https://documentation.solarwinds.com/en/success_center/kss/content/kss_adminguide_syslog_protocol.htm

var rfc3164parts = [...]partType{
	{rfcFormatKey, "string"}, // Non-RFC: Added by syslogsidecar
	{"priority", "int"},
	{"facility", "int"},
	{severityKey, "int"},
	{"timestamp", "time.Time"},
	{"hostname", "string"},
	{rfc3164OnlyKey, "string"},
	{"content", "string"},
}

// RFC5424 parameters with type
// https://hackmd.io/@njjack/syslogformat
var rfc5424parts = [...]partType{
	{rfcFormatKey, "string"}, // Non-RFC: Added by syslogsidecar
	{"priority", "int"},
	{"facility", "int"},
	{severityKey, "int"},
	{"version", "int"},
	{"timestamp", "time.Time"},
	{"hostname", "string"},
	{"app_name", "string"},
	{"proc_id", "string"},
	{"msg_id", "string"},
	{rfc5424OnlyKey, "string"},
	{"message", "string"},
}

// Former message - for badly formatted syslog message
var formerMessage = [...]partType{
	{Formermessage, "string"},
}

type syslogmsgparts struct {
	parts
}

func (mp *syslogmsgparts) pack(logParts format.LogParts, err error) error {

	if mp == nil {
		return fmt.Errorf("nil syslogmsgparts")
	}

	if logParts == nil {
		return fmt.Errorf("nil logParts")
	}

	if len(logParts) == 0 {
		return fmt.Errorf("empty logParts")
	}

	if err != nil {
		mp.packParts(formerMessage[:], logParts)
		return nil
	}

	if _, exists := logParts[rfc5424OnlyKey]; exists {
		logParts[rfcFormatKey] = rfc5424
		mp.packParts(rfc5424parts[:], logParts)
		return nil
	}

	if _, exists := logParts[rfc3164OnlyKey]; exists {
		logParts[rfcFormatKey] = rfc3164
		mp.packParts(rfc3164parts[:], logParts)
		return nil
	}
	mp.packParts(formerMessage[:], logParts)
	return nil
}

func (mp *syslogmsgparts) packParts(parts []partType, logParts format.LogParts) {

	mp.set(128)

	count := len(parts)
	mp.setRuneAt(0, rune(count))
	mp.skip(count + 1)

	for i, part := range parts {
		v, exists := logParts[part.name]

		if !exists {
			mp.setRuneAt(i+1, 0)
			continue
		}

		mp.setRuneAt(i+1, rune(mp.appendText(toString(v, part.kind))))
	}
}

func (mp *syslogmsgparts) Unpack(put func(name, value string) error) error {

	if mp == nil {
		return fmt.Errorf("nil syslogmsgparts")
	}

	if len(mp.data) == 0 {
		return fmt.Errorf("empty syslogmsgparts")
	}

	count, _ := mp.runeAt(0)

	switch int(count) {
	case badMessageParts:
		return mp.unpackParts(formerMessage[:], put)
	case rfc5424Parts:
		return mp.unpackParts(rfc5424parts[:], put)
	case rfc3164Parts:
		return mp.unpackParts(rfc3164parts[:], put)
	}

	return fmt.Errorf("Wrong packed syslog message")
}

func (mp *syslogmsgparts) unpackParts(parts []partType, put func(name, value string) error) error {
	mp.rewind()
	count, _ := mp.runeAt(0)
	mp.skip(int(count + 1))

	for i, part := range parts {
		vlen, _ := mp.runeAt(1 + i)
		value, err := mp.part(int(vlen))
		if err != nil {
			return err
		}
		err = put(part.name, value)
		if err != nil {
			return err
		}
	}

	return nil
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
		result = tval.Format(time.RFC3339)
		return result
	}

	return result
}

//////////////////////////////////////////////////////////////////////

func RFC3164Props() map[string]string {
	return map[string]string{
		"priority":     "int",
		"facility":     "int",
		severityKey:    "int",
		"timestamp":    "time.Time",
		"hostname":     "string",
		rfc3164OnlyKey: "string",
		"content":      "string",
	}
}

var rfc3164props = RFC3164Props()

var rfc3164names = [7]string{
	"priority", "facility", severityKey, "timestamp", "hostname", rfc3164OnlyKey, "content"}

func RFC5424Props() map[string]string {
	return map[string]string{
		"priority":     "int",
		"facility":     "int",
		severityKey:    "int",
		"version":      "int",
		"timestamp":    "time.Time",
		"hostname":     "string",
		"app_name":     "string",
		"proc_id":      "string",
		"msg_id":       "string",
		rfc5424OnlyKey: "string",
		"message":      "string",
	}
}

var rfc5424props = RFC5424Props()

var rfc5424names = [11]string{
	"priority", "facility", severityKey, "version", "timestamp", "hostname",
	"app_name", "proc_id", "msg_id", rfc5424OnlyKey, "message"}

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

	if _, exists := logParts[rfc5424OnlyKey]; exists {
		return toRFC5424(logParts)
	}

	if _, exists := logParts[rfc3164OnlyKey]; exists {
		return toRFC3164(logParts)
	}

	return msgFromFormerMsg(logParts)

}

func msgFromFormerMsg(logParts format.LogParts) sputnik.Msg {
	msg := make(sputnik.Msg)
	formerMsg := logParts[Formermessage].(string)
	msg[Formermessage] = formerMsg
	return msg
}

func toRFC5424(logParts format.LogParts) sputnik.Msg {
	msg := make(sputnik.Msg)
	msg[rfcFormatKey] = rfc5424

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

func toRFC3164(logParts format.LogParts) sputnik.Msg {
	msg := make(sputnik.Msg)
	msg[rfcFormatKey] = rfc3164

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

//////////////////////////////////////////////////////////////////////
