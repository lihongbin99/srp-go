package p2p

import (
	"fmt"
	"srp-go/common/io"
	"srp-go/common/msg"
	"srp-go/common/utils"
	"sync"
)

type clientHandle struct {
	serverTCP      *io.TCP
	configsTCP     map[string]*clientConfigTCP
	connectTCP     map[utils.ID]*clientConnectTCP
	connectTCPLock sync.Mutex
}

func GetNewClientHandle() *clientHandle {
	return &clientHandle{
		nil,
		make(map[string]*clientConfigTCP),
		make(map[utils.ID]*clientConnectTCP),
		sync.Mutex{},
	}
}

func (t *clientHandle) Start(serverTCP *io.TCP) {
	t.serverTCP = serverTCP
	// 开始服务
	go t.startService()
}

func (t *clientHandle) Handle(serverTCP *io.TCP, message msg.Message) (do bool) {
	do = true
	switch m := message.(type) {
	case *msg.P2pNewConnectRequest:
		if m.Protocol == "tcp" {
			go t.p2pNewConnectRequestTCP(serverTCP, m)
		} else {
			_ = serverTCP.WriteSecurityMessage(&msg.P2pNewConnectResponse{
				ServerId: m.ServerId,
				Protocol: m.Protocol,
				Result:   fmt.Sprintf("protocol error: %s", m.Protocol),
			})
		}
	default:
		do = false
	}
	return
}

func (t *clientHandle) Close() {
	// 关闭服务
	for _, c := range t.configsTCP {
		if c.listen != nil {
			_ = c.listen.Close()
		}
	}
	t.serverTCP = nil
}

func (t *clientHandle) ConnectInfo() string {
	connects := ""
	for _, ct := range t.connectTCP {
		connects += "		" + ct.String() + ",\n"
	}
	return "	p2p[\n" +
		connects +
		"	]\n"
}
