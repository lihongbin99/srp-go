package nat

import (
	"fmt"
	"srp-go/common/io"
	"srp-go/common/msg"
)

func (t *serverHandle) register(tcp *io.TCP, message *msg.NatRegisterRequest) {
	// 开启端口监听
	if message.Protocol == "tcp" {
		t.registerTCP(tcp, message)
	} else if message.Protocol == "udp" {
		t.registerUDP(tcp, message)
	} else {
		_ = tcp.WriteSecurityMessage(&msg.NatRegisterResponse{
			ServiceName: message.ServiceName,
			Result:      fmt.Sprintf("protocol error: %s", message.Protocol),
		})
	}
}

func (t *serverHandle) newConnectResponse(tcp *io.TCP, message *msg.NatNewConnectResponse) {
	if message.Protocol == "tcp" {
		t.newConnectResponseTCP(tcp, message)
	} else if message.Protocol == "udp" {
		t.newConnectResponseUDP(tcp, message)
	} else {
		log.Error("newConnectResponse Protocol error", message.Protocol)
	}
}
