package main

import (
	"fmt"
	"srp-go/common/io"
	"srp-go/common/msg"
	"srp-go/core"
)

func notifyRequest(tcp *io.TCP, message *msg.NotifyRequest) {
	core.TcpMapLock.Lock()
	defer core.TcpMapLock.Unlock()
	message.OperateName = tcp.ClientName
	clientTCP, ok := core.TcpMap[message.ClientName]
	if !ok {
		_ = tcp.WriteSecurityMessage(&msg.NotifyResponse{Result: fmt.Sprintf("no find client name by %s", message.ClientName)})
		return
	}

	_ = clientTCP.WriteSecurityMessage(message)
}

func notifyResponse(_ *io.TCP, message *msg.NotifyResponse) {
	core.TcpMapLock.Lock()
	defer core.TcpMapLock.Unlock()
	clientTCP, ok := core.TcpMap[message.OperateName]
	if !ok {
		log.Error("notify response no find operate name", message.OperateName)
		return
	}

	_ = clientTCP.WriteSecurityMessage(message)
}
