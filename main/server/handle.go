package main

import (
	"net"
	"srp-go/common/io"
	"srp-go/common/msg"
)

type ServerHandle interface {
	NewConnect(tcp *io.TCP, connectType msg.ClientType) (do bool)
	HandleTCP(tcp *io.TCP, message msg.Message) (do, re bool)
	HandleUDP(udp *io.UDP, message msg.Message, addr *net.UDPAddr) (do bool)
	Close(tcp *io.TCP)

	ConnectInfo() string
}
