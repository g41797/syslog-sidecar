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

func newsyslogmsgparts() *syslogmsgparts {
	result := new(syslogmsgparts)
	result.set(128)
	return result
}

func (mp *syslogmsgparts) pack(logParts format.LogParts) error {

	if mp == nil {
		return fmt.Errorf("nil syslogmsgparts")
	}

	if logParts == nil {
		return fmt.Errorf("nil logParts")
	}

	if len(logParts) == 0 {
		return fmt.Errorf("empty logParts")
	}

	if _, exists := logParts[Formermessage]; exists {
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

func toMsg(logParts format.LogParts) sputnik.Msg {

	if logParts == nil {
		return nil
	}

	if len(logParts) == 0 {
		return nil
	}

	msg := Get()

	slm, _ := msg[syslogmessage].(*syslogmsgparts)

	perr := slm.pack(logParts)
	if perr != nil {
		return nil
	}

	return msg

}

func (mp *syslogmsgparts) priority() (string, error) {
	mp.rewind()

	count, _ := mp.runeAt(0)

	if int(count) <= badMessageParts {
		return Formermessage, fmt.Errorf("non rfc message")
	}

	rfclen, _ := mp.runeAt(1)

	mp.skip(int(count + rfclen + 1))

	prlen, _ := mp.runeAt(2)

	return mp.part(int(prlen))
}
