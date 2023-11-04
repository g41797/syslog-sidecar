package syslogsidecar

import (
	"github.com/g41797/sputnik"
)

const (
	ReceiverName           = "syslogreceiver"
	ReceiverResponsibility = "syslogreceiver"

	ProducerName           = "syslogproducer"
	ProducerResponsibility = "syslogproducer"

	WriterName           = "syslogwriter"
	WriterResponsibility = "syslogwriter"
)

func receiverDescriptor() sputnik.BlockDescriptor {
	return sputnik.BlockDescriptor{Name: ReceiverName, Responsibility: ReceiverResponsibility}
}

func init() {
	sputnik.RegisterBlockFactory(ReceiverName, receiverBlockFactory)
}

func receiverBlockFactory() *sputnik.Block {
	receiver := new(receiver)
	block := sputnik.NewBlock(
		sputnik.WithInit(receiver.init),
		sputnik.WithRun(receiver.run),
		sputnik.WithFinish(receiver.finish),
	)
	return block
}

type receiver struct {
	conf     SyslogConfiguration
	syslogd  *server
	producer sputnik.BlockCommunicator

	// Used for synchronization
	// between finish and run
	stop chan struct{}
	done chan struct{}
}

// Init
func (rcv *receiver) init(fact sputnik.ConfFactory) error {
	if err := fact(ReceiverName, &rcv.conf); err != nil {
		return err
	}

	syslogd := newServer(rcv.conf)

	if err := syslogd.initServer(); err != nil {
		return err
	}

	syslogd.setupHandling(nil)

	rcv.syslogd = syslogd
	rcv.stop = make(chan struct{}, 1)

	return nil
}

// Finish:
func (rcv *receiver) finish(init bool) {
	if init {
		rcv.stopSyslog()
		return
	}

	close(rcv.stop) // Cancel Run

	<-rcv.done // Wait finish of Run
	return
}

// Run:
func (rcv *receiver) run(bc sputnik.BlockCommunicator) {

	err := rcv.syslogd.start()
	if err != nil {
		panic(err)
	}

	defer rcv.stopSyslog()

	rcv.done = make(chan struct{})
	defer close(rcv.done)

	producer, exists := bc.Communicator(ProducerResponsibility)
	if !exists {
		panic("Syslog producer block does not exists")
	}

	rcv.producer = producer
	rcv.syslogd.setupHandling(rcv.producer)

	<-rcv.stop

	return
}

func (rcv *receiver) stopSyslog() {

	if rcv == nil {
		return
	}

	if rcv.syslogd == nil {
		return
	}

	rcv.syslogd.setupHandling(nil)
	rcv.syslogd.stop()

	return
}
