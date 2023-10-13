package main

import (

	// sputnik framework:
	"github.com/g41797/sputnik/sidecar"
	//
	//	syslogsidecar blocks: receiver|producer|client|consumer
	_ "github.com/g41797/syslogsidecar"
	//
	//	eventBus plugins:connector|msgconsumer|msgproducer
	"github.com/g41797/syslogsidecar/internal/plugins"
)

func main() {
	sidecar.Start(new(plugins.BrokerConnector))
}
