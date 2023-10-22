package syslogsidecar

import (
	"sync"

	"github.com/g41797/sputnik"
)

func Get() sputnik.Msg {
	return mPool.Get().(sputnik.Msg)
}

func Put(msg sputnik.Msg) {
	mPool.Put(msg)
}

var mPool = sync.Pool{New: newMessage}

const syslogmessage = "syslogmessage"

func newMessage() interface{} {
	msg := make(sputnik.Msg)
	syslogmsgparts := new(syslogmsgparts)
	syslogmsgparts.set(128)
	msg[syslogmessage] = syslogmsgparts
	return msg
}
