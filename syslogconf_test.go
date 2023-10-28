package syslogsidecar

import "testing"

func Test_GetTarget(t *testing.T) {

	for _, test := range tests {
		run(test, t)
	}

}

var tests = [][]any{
	// selector target t/f facility severity t/f exptarget
	{"notice,warning", "folder4", false, "ftp", "warning", false, "folder4"},
	{"notice,notice", "mailfolder", false, "ftp", "warning", false, ""},
	{"notice,mail", "mailfolder", true},
	{"mail.notice,warning", "mailfolder", false, "mail", "warning", false, "mailfolder"},
	{"mail.notice", "ftpfolder", false, "ftp", "notice", false, ""},
	{"mail,ftp", "ftpfolder", false, "ftp", "notice", false, "ftpfolder"},
	{"mail,mail", "anyfolder", false, "ftp", "notice", false, ""},
	{"mail,data", "anyfolder", true},
	{"data", "anyfolder", false, "data", "", false, "anyfolder"},
	{"data", "anyfolder", false, "data", "crit", true},
	{"data", "anyfolder", false, "ftp", "crit", false, ""},
	{"any", "", true},
	{"", "", true},
}

const (
	selindex               = 0
	targindex              = 1
	tofindshouldfailindex  = 2
	facilindex             = 3
	severindex             = 4
	gettargshouldfailindex = 5
	exptargetindex         = 6
)

func run(params []any, t *testing.T) {
	selector := params[selindex].(string)
	target := params[targindex].(string)
	slfEntry := slfEntry{selector, target}

	tf, err := slfEntry.toFinder()

	shouldfail := params[tofindshouldfailindex].(bool)

	if shouldfail && (err == nil) {
		t.Errorf("toFinder should fail for %s %s", selector, target)
		return
	}

	if !shouldfail && (err != nil) {
		t.Errorf("toFinder should not fail for %s %s. error - %v", selector, target, err)
		return
	}

	if err != nil {
		return
	}

	if tf == nil {
		t.Errorf("toFinder should be not nil for %s %s", selector, target)
		return
	}

	facility := params[facilindex].(string)
	severity := params[severindex].(string)

	targ, ok := tf.gettarget(facility, severity)
	shouldfail = params[gettargshouldfailindex].(bool)

	if shouldfail && ok {
		t.Errorf("gettarget should fail for %s %s", facility, severity)
		return
	}

	if !shouldfail && !ok {
		t.Errorf("gettarget should not fail for %s %s", facility, severity)
		return
	}

	if shouldfail {
		return
	}

	exptarg := params[exptargetindex].(string)
	if targ != exptarg {
		t.Errorf("actual target %s != expected target%s", targ, exptarg)
	}
}
