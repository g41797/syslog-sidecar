package plugins

import (
	"github.com/asaskevich/EventBus"
	"github.com/g41797/sputnik"
)

var _ sputnik.ServerConnector = &BrokerConnector{}

type BrokerConnector struct {
	ebus EventBus.Bus
}

func (c *BrokerConnector) Connect(cf sputnik.ConfFactory) (conn sputnik.ServerConnection, err error) {
	if !c.IsConnected() {
		c.ebus = EventBus.New()
	}

	return c.ebus, nil

}

func (c *BrokerConnector) IsConnected() bool {
	if c == nil {
		return false
	}

	if c.ebus == nil {
		return false
	}

	return true
}

func (c *BrokerConnector) Disconnect() {
	c.ebus = nil
	return
}
