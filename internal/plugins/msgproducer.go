package plugins

import (
	"fmt"

	"github.com/asaskevich/EventBus"
	"github.com/g41797/sputnik"
	"github.com/g41797/sputnik/sidecar"
	"github.com/g41797/syslogsidecar"
)

func init() {
	syslogsidecar.RegisterMessageProducerFactory(newMsgProducer)
}

const MsgProducerConfigName = syslogsidecar.ProducerName

type MsgPrdConfig struct {
	TOPIC string
}

func newMsgProducer() sidecar.MessageProducer {
	return &msgProducer{}
}

type msgProducer struct {
	conf MsgPrdConfig
	ebus EventBus.Bus
}

func (mpr *msgProducer) Connect(cf sputnik.ConfFactory, evBus sputnik.ServerConnection) error {
	err := cf(MsgProducerConfigName, &mpr.conf)
	if err != nil {
		return err
	}

	mpr.ebus = evBus.(EventBus.Bus)

	return nil
}

func (mpr *msgProducer) Disconnect() {
	mpr.ebus = nil
	return
}

func (mpr *msgProducer) Produce(msg sputnik.Msg) error {
	if mpr.ebus == nil {
		return fmt.Errorf("connection with broker does not exist")
	}

	if !mpr.ebus.HasCallback(mpr.conf.TOPIC) {
		return fmt.Errorf("subscriber for topic %s does not exist", mpr.conf.TOPIC)
	}

	props, err := syslogsidecar.UnpackToMap(msg)

	syslogsidecar.Put(msg)

	if err != nil {
		return err
	}

	mpr.ebus.Publish(mpr.conf.TOPIC, props)

	return nil
}
