package main

import (
	"srp-go/common/io"
	"srp-go/common/msg"
)

type ClientHandle interface {
	Start(serverTCP *io.TCP)
	Handle(tcp *io.TCP, message msg.Message) (do bool)
	Close()

	ConnectInfo() string
}
