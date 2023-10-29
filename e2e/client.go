package e2e

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/RackSec/srslog"
	"github.com/RoaringBitmap/roaring"
	"github.com/g41797/sputnik"
	"github.com/g41797/syslogsidecar"
)

const (
	syslogClientName           = "syslogclient"
	syslogClientResponsibility = "syslogclient"

	rfc3164 = "RFC3164"
	rfc5424 = "RFC5424"

	rfcFormatKey = "rfc"
)

func syslogClientDescriptor() sputnik.BlockDescriptor {
	return sputnik.BlockDescriptor{Name: syslogClientName, Responsibility: syslogClientResponsibility}
}

func init() {
	sputnik.RegisterBlockFactory(syslogClientName, syslogClientBlockFactory)
}

func syslogClientBlockFactory() *sputnik.Block {
	client := new(client)
	block := sputnik.NewBlock(
		sputnik.WithInit(client.init),
		sputnik.WithRun(client.run),
		sputnik.WithFinish(client.finish),
		sputnik.WithOnMsg(client.processBrokerMsg),
	)
	return block
}

const max_LOG_MESSAGES = 50 // was 1000000

type client struct {
	conf    syslogsidecar.SyslogConfiguration
	loggers []*srslog.Writer
	flip    bool
	bc      sputnik.BlockCommunicator

	started   bool
	startTime time.Time
	currIndx  int
	processed *roaring.Bitmap
	successN  int
	recvN     int

	startFlow  chan struct{}
	stopFlow   chan struct{}
	msgheaders chan map[string]string
	nextSend   chan struct{}

	stop chan struct{}
	done chan struct{}
}

// Init
func (cl *client) init(fact sputnik.ConfFactory) error {
	if err := fact(syslogsidecar.ReceiverName, &cl.conf); err != nil {
		return err
	}

	cl.loggers = make([]*srslog.Writer, 2)
	cl.stop = make(chan struct{}, 1)
	cl.done = make(chan struct{}, 1)
	cl.startFlow = make(chan struct{}, 1)
	cl.stopFlow = make(chan struct{}, 1)
	cl.msgheaders = make(chan map[string]string, 1)
	cl.nextSend = make(chan struct{}, 1)

	return nil
}

// Finish:
func (cl *client) finish(init bool) {
	if init {
		return
	}

	close(cl.stop) // Cancel Run

	<-cl.done // Wait finish of Run
	return
}

// Run
func (cl *client) run(bc sputnik.BlockCommunicator) {

	cl.bc = bc

	defer close(cl.done)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

loop:
	for {
		select {
		case <-cl.stop:
			return
		case <-ticker.C:
			if err := cl.openLoggers(); err == nil {
				break loop
			}
		}
	}

	cl.runLoop()

	cl.closeLoggers()
	cl.report()

	return
}

func (cl *client) runLoop() {

	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case headers := <-cl.msgheaders:
			cl.update(headers)
		case <-cl.nextSend:
			cl.sendNext()
		case <-cl.stop:
			return
		case <-cl.startFlow:
			cl.startflow()
		case <-cl.stopFlow:
			cl.stopflow()
		case <-ticker.C:
			cl.report()
		}
	}
}

// OnMsg:
func (cl *client) processBrokerMsg(brokermsg sputnik.Msg) {
	if brokermsg == nil {
		return
	}

	name, exists := brokermsg["name"]
	if !exists {
		return
	}

	switch name {
	case "start":
		cl.startFlow <- struct{}{}
	case "stop":
		cl.stopFlow <- struct{}{}
	case "consumed":
		headers, ok := brokermsg["consumed"].(map[string]string)
		if ok && headers != nil {
			cl.msgheaders <- headers
		}
	}

	return
}

func (cl *client) startflow() {
	if cl.started {
		return
	}
	cl.startTime = time.Now()
	cl.started = true
	cl.processed = roaring.New()
	cl.sendNext()
}

func (cl *client) sendNext() {
	if !cl.started {
		return
	}

	if cl.currIndx >= max_LOG_MESSAGES {
		return
	}

	var err error
	if cl.flip {
		err = cl.loggers[cl.currIndx%2].Warning(strconv.Itoa(cl.currIndx))
	} else {
		err = cl.loggers[cl.currIndx%2].Emerg(strconv.Itoa(cl.currIndx))
	}

	cl.flip = !cl.flip

	if err != nil {
		cl.stopflow()
		return
	}

	cl.currIndx++
	cl.nextSend <- struct{}{}

	return
}

func (cl *client) stopflow() {
	cl.started = false
	sendQuit()
}

func (cl *client) update(hdrs map[string]string) {
	if hdrs == nil {
		return
	}
	cl.recvN++

	rfc, ok := hdrs[rfcFormatKey]
	if !ok {
		return
	}

	valName := "message"

	if rfc == rfc3164 {
		valName = "content"
	}

	msgText, ok := hdrs[valName]
	if !ok {
		return
	}

	msgIndex, err := strconv.Atoi(msgText)
	if err != nil {
		return
	}

	if msgIndex >= max_LOG_MESSAGES {
		return
	}

	if msgIndex > cl.currIndx {
		return
	}

	if wasAdded := cl.processed.CheckedAdd(uint32(msgIndex)); !wasAdded {
		return
	}

	cl.successN++

	if cl.successN >= max_LOG_MESSAGES {
		cl.stopflow()
	}

	return
}

func (cl *client) report() {
	if cl.currIndx > 0 {
		elp := time.Since(cl.startTime).String()
		fmt.Printf("\n\n\t\t%s   Was send %d messages. Successfully consumed %d Received %d\n\n", elp, cl.currIndx, cl.successN, cl.recvN)
	}
	return
}

func (cl *client) openLoggers() error {
	lgr, err := newLogWriter(cl.conf, srslog.RFC3164Formatter)
	if err != nil {
		return err
	}
	cl.loggers[0] = lgr

	lgr, err = newLogWriter(cl.conf, srslog.RFC5424Formatter)
	if err != nil {
		cl.loggers[0].Close()
		cl.loggers[0] = nil
		return err
	}
	cl.loggers[1] = lgr
	return nil
}

func (cl *client) closeLoggers() {
	for _, lgr := range cl.loggers {
		if lgr != nil {
			lgr.Close()
		}
	}
}

func newLogWriter(cnf syslogsidecar.SyslogConfiguration, rfcForm srslog.Formatter) (*srslog.Writer, error) {
	w, err := srslog.Dial("tcp", cnf.ADDRTCP, srslog.LOG_ALERT, "re2e")
	if err != nil {
		return nil, err
	}
	w.SetFormatter(rfcForm)
	return w, nil
}

func sendQuit() error {
	cproc, err := os.FindProcess(os.Getpid())
	if err != nil {
		return err
	}
	err = cproc.Signal(os.Interrupt)
	return err
}
