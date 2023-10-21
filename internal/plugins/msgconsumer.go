package plugins

import (
	"github.com/asaskevich/EventBus"
	"github.com/g41797/sputnik"
	"github.com/g41797/sputnik/sidecar"
	"github.com/g41797/syslogsidecar"
	"github.com/g41797/syslogsidecar/e2e"
)

func init() {
	e2e.RegisterMessageConsumerFactory(newMsgConsumer)
}

const MsgConsumerConfigName = MsgProducerConfigName

type msgConsumer struct {
	conf    MsgPrdConfig
	ebus    EventBus.Bus
	sender  sputnik.BlockCommunicator
	started bool
}

func newMsgConsumer() sidecar.MessageConsumer {
	return new(msgConsumer)
}

func (mcn *msgConsumer) Connect(cf sputnik.ConfFactory, evBus sputnik.ServerConnection) error {
	err := cf(MsgProducerConfigName, &mcn.conf)
	if err != nil {
		return err
	}

	mcn.ebus = evBus.(EventBus.Bus)

	return nil
}

func (cons *msgConsumer) Consume(sender sputnik.BlockCommunicator) error {
	if cons.started {
		return nil
	}

	cons.sender = sender

	err := cons.ebus.SubscribeAsync(cons.conf.TOPIC, cons.onMessage, true)
	if err != nil {
		return err
	}

	cons.startTest()
	cons.started = true
	return nil
}

func (cons *msgConsumer) onMessage(inmsg map[string]string) {

	if inmsg == nil {
		return
	}

	if cons.sender == nil {
		return
	}

	omsg := ConvertConsumeMsg(inmsg)

	if omsg == nil {
		return
	}
	cons.sender.Send(omsg)
	return
}

func (cons *msgConsumer) Disconnect() {
	if cons == nil {
		return
	}

	if !cons.started {
		return
	}

	cons.ebus.Unsubscribe(cons.conf.TOPIC, cons.onMessage)

	cons.stopTest()
	cons.started = false
	return
}

func ConvertConsumeMsg(inmsg map[string]string) sputnik.Msg {
	if inmsg == nil {
		return nil
	}

	smsg := sputnik.Msg{}
	smsg["name"] = "consumed"
	smsg["consumed"] = inmsg
	smsg[syslogsidecar.Formermessage] = ""

	return smsg
}

func (cons *msgConsumer) startTest() {
	if cons.sender == nil {
		return
	}
	msg := sputnik.Msg{}
	msg["name"] = "start"
	cons.sender.Send(msg)
}

func (cons *msgConsumer) stopTest() {
	if cons.sender == nil {
		return
	}
	msg := sputnik.Msg{}
	msg["name"] = "stop"
	cons.sender.Send(msg)
}
