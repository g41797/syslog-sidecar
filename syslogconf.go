package syslogsidecar

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/g41797/sputnik"
	"github.com/g41797/sputnik/sidecar"
)

// Returns list of non-repeating "targets" for the message according to facility and severity
// of the message and content of syslogconf.json file.
// Usually error returned for the case of absent or wrong syslogconf.json file.
// nil, nil - means no defined targets for the message.
// Decision for this case on producer, e.g. use default target(topic, station, etc)
// Sidecar transfers targets to producer with solely processing -
// trim spaces on both sides of the string.
// Target may be any non-empty valid for JSON format string.
func Targets(msg sputnik.Msg) ([]string, error) {

	bfonce.Do(buildFinders)

	if tfError != nil {
		return nil, tfError
	}

	if msg == nil {
		return nil, fmt.Errorf("nil msg")
	}

	slm, exists := msg[syslogmessage]
	if !exists {
		return nil, fmt.Errorf("empty msg")
	}

	syslogmsgparts, ok := slm.(*syslogmsgparts)
	if !ok {
		return nil, fmt.Errorf("wrong msg")
	}

	priority, err := syslogmsgparts.priority()

	if err != nil {
		return nil, err
	}

	facility, severiry := facsev(priority)

	var targets []string

	trgmap := make(map[string]bool)

	for _, finder := range tFinders {
		target, _ := finder.gettarget(facility, severiry)
		if len(target) != 0 {
			if _, exists := trgmap[target]; !exists {
				trgmap[target] = true
				targets = append(targets, target)
			}
		}
	}

	return targets, nil
}

// Returns list of all non-repeating "targets" existing in syslogconf.json file
// and error for absent or wrong syslogconf.json file.
func AllTargets() ([]string, error) {

	bfonce.Do(buildFinders)

	if tfError != nil {
		return nil, tfError
	}

	if len(tFinders) == 0 {
		return nil, fmt.Errorf("empty list of finders")
	}

	var targets []string

	trgmap := make(map[string]bool)

	for _, finder := range tFinders {
		target := finder.target
		if _, exists := trgmap[target]; !exists {
			trgmap[target] = true
			targets = append(targets, target)
		}
	}

	return targets, nil
}

var tFinders []*targetFinder
var tfError error
var bfonce sync.Once

func buildFinders() {
	tFinders, tfError = buildfinders()
}

func buildfinders() (finders []*targetFinder, fErr error) {

	slfEntries, slfError := readsyslogconf()

	if slfError != nil {
		fErr = slfError
		return
	}

	if len(slfEntries) == 0 {
		fErr = fmt.Errorf("empty syslogconf file")
		return
	}

	var tfinders []*targetFinder

	for _, entry := range slfEntries {
		tf, err := entry.toFinder()
		if err != nil {
			fErr = err
			return
		}
		tfinders = append(tfinders, tf)
	}

	return tfinders, nil
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
			if len(sev) == 0 {
				continue
			}
			if !isSeverity(sev) {
				return nil, fmt.Errorf("wrong severity %s", sev)
			}
			_, exists := smap[sev]
			if !exists {
				tf.severities = append(tf.severities, sev)
			}
		}

		if len(tf.severities) == 0 {
			tf.gettarget = tf.listoffacilities
		} else {
			tf.gettarget = tf.severitiesOfFacitity
		}

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
