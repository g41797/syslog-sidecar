package syslogsidecar

import "strconv"

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

	prval, _ := strconv.Atoi(priority)

	facility, _ = fis[prval/8]
	severiry, _ = sis[prval%8]

	return
}

var fsi = swap(fis)
var ssi = swap(sis)

func swap(in map[int]string) map[string]int {
	result := make(map[string]int)

	for key, val := range fis {
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
