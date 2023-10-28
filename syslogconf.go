package syslogsidecar

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/g41797/sputnik/sidecar"
)

type slfEntry struct {
	Selector string
	Target   string
}

type targetFinder struct {
	facilities []string
	severities []string
	target     string
	gettarget  func(facility, severiry string) (string, bool)
}

func (tf *targetFinder) data(facility, severity string) (string, bool) {
	if (facility == tf.facilities[0]) && (len(severity) == 0) {
		return tf.target, true
	}

	if !isFacility(facility) {
		return "", false
	}

	if !isSeverity(severity) {
		return "", false
	}

	return "", true
}

func (tf *targetFinder) listoffacilities(facility, severity string) (string, bool) {
	if !isFacility(facility) {
		return "", false
	}

	if !isSeverity(severity) {
		return "", false
	}

	for _, fac := range tf.facilities {
		if facility == fac {
			return tf.target, true
		}
	}

	return "", true
}

func (tf *targetFinder) severities_(facility, severity string) (string, bool) {
	if !isFacility(facility) {
		return "", false
	}

	if !isSeverity(severity) {
		return "", false
	}

	for _, sev := range tf.severities {
		if severity == sev {
			return tf.target, true
		}
	}

	return "", true
}

func (tf *targetFinder) severitiesOfFacitity(facility, severity string) (string, bool) {
	if !isFacility(facility) {
		return "", false
	}

	if !isSeverity(severity) {
		return "", false
	}

	if facility != tf.facilities[0] {
		return "", true
	}
	return tf.severities_(facility, severity)
}

func (se *slfEntry) toFinder() (*targetFinder, error) {

	fmap := make(map[string]bool)
	smap := make(map[string]bool)

	tf := new(targetFinder)
	tf.target = se.Target

	// data
	if se.Selector == Formermessage {
		tf.facilities = append(tf.facilities, se.Selector)
		tf.gettarget = tf.data
		return tf, nil
	}

	// f.s1,s2,...sN
	before, after, found := strings.Cut(se.Selector, ".")

	if found {
		if !isFacility(before) {
			return nil, fmt.Errorf("wrong facility %s", before)
		}
		_, exists := fmap[before]
		if !exists {
			tf.facilities = append(tf.facilities, before)
		}

		if len(after) == 0 {
			tf.gettarget = tf.listoffacilities
			return tf, nil
		}

		// s1,s2,....sN
		severities := strings.Split(after, ",")

		for _, sev := range severities {
			if !isSeverity(sev) {
				return nil, fmt.Errorf("wrong severity %s", sev)
			}
			_, exists := smap[sev]
			if !exists {
				tf.severities = append(tf.severities, sev)
			}
		}

		tf.gettarget = tf.severitiesOfFacitity
		return tf, nil
	}
	// s1,s2,....sN or f1,f2,...fN ?

	list := strings.Split(before, ",")

	if isSeverity(list[0]) {

		for _, sev := range list {
			if !isSeverity(sev) {
				return nil, fmt.Errorf("wrong severity %s", sev)
			}
			_, exists := smap[sev]
			if !exists {
				tf.severities = append(tf.severities, sev)
			}
		}

		tf.gettarget = tf.severities_
		return tf, nil
	}

	for _, fac := range list {
		if !isFacility(fac) {
			return nil, fmt.Errorf("wrong facility %s", fac)
		}
		_, exists := fmap[fac]
		if !exists {
			tf.facilities = append(tf.facilities, fac)
		}
	}

	tf.gettarget = tf.listoffacilities
	return tf, nil
}

func readsyslogconf() ([]slfEntry, error) {

	confFolder, err := sidecar.ConfFolder()

	if err != nil {
		return nil, err
	}

	fPath := filepath.Join(confFolder, "syslogconf.json")

	return slogconfbypath(fPath)
}

func slogconfbypath(fPath string) ([]slfEntry, error) {

	entriesRaw, err := os.ReadFile(fPath)
	if err != nil {
		return nil, err
	}

	var entries []slfEntry

	json.Unmarshal([]byte(entriesRaw), &entries)

	for _, entry := range entries {

		if len(entry.Selector) > 0 {
			entry.Selector = strings.ToLower(strings.ReplaceAll(entry.Selector, " ", ""))
		}

		if len(entry.Selector) == 0 {
			return nil, fmt.Errorf("empty selector")
		}

		if len(entry.Target) > 0 {
			entry.Target = strings.TrimSpace(entry.Target)
		}

		if len(entry.Target) == 0 {
			return nil, fmt.Errorf("empty target")
		}

	}

	return entries, nil
}

var fis = map[int]string{
	0:  "kern",
	1:  "user",
	2:  "mail",
	3:  "daemon",
	4:  "auth",
	5:  "syslog",
	6:  "lpr",
	7:  "news",
	8:  "uucp",
	9:  "cron",
	10: "authpriv",
	11: "ftp",
	16: "local0",
	17: "local1",
	18: "local2",
	19: "local3",
	20: "local4",
	21: "local5",
	22: "local6",
	23: "local7",
}

var sis = map[int]string{
	0: "emerg",
	1: "alert",
	2: "crit",
	3: "err",
	4: "warning",
	5: "notice",
	6: "info",
	7: "debug",
}

func facsev(priority string) (facility, severiry string) {
	if len(priority) == 0 {
		return "", ""
	}

	if priority == Formermessage {
		return Formermessage, ""
	}

	prval, _ := strconv.Atoi(priority)

	facility, _ = fis[prval/8]
	severiry, _ = sis[prval%8]

	return
}

var fsi = swap(fis)
var ssi = swap(sis)

func swap(in map[int]string) map[string]int {
	result := make(map[string]int)

	for key, val := range in {
		result[val] = key
	}

	return result
}

func isFacility(fac string) bool {
	if len(fac) == 0 {
		return false
	}

	_, ok := fsi[fac]

	return ok
}

func isSeverity(sev string) bool {
	if len(sev) == 0 {
		return false
	}

	_, ok := ssi[sev]

	return ok
}
