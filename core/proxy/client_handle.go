package proxy

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
	configsUDP     map[string]*clientConfigUDP
	connectTCP     map[utils.ID]*clientConnectTCP
	connectTCPLock sync.Mutex
	connectUDP     map[utils.ID]*clientConnectUDP
	connectUDPLock sync.Mutex
	udpMap         map[string]utils.ID
	udpMapLock     sync.Mutex
}

func GetNewClientHandle() *clientHandle {
	return &clientHandle{
		nil,
		make(map[string]*clientConfigTCP),
		make(map[string]*clientConfigUDP),
		make(map[utils.ID]*clientConnectTCP),
		sync.Mutex{},
		make(map[utils.ID]*clientConnectUDP),
		sync.Mutex{},
		make(map[string]utils.ID),
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
	case *msg.ProxyNewConnectRequest:
		if m.Protocol == "tcp" {
			go t.proxyNewConnectRequestTCP(serverTCP, m)
		} else if m.Protocol == "udp" {
			go t.proxyNewConnectRequestUDP(serverTCP, m)
		} else {
			_ = serverTCP.WriteSecurityMessage(&msg.ProxyNewConnectResponse{
				ServerId: m.ServerId,
				Protocol: m.Protocol,
				Result:   fmt.Sprintf("protocol error: %s", m.Protocol),
			})
		}
	case *msg.ProxyNewConnectResponse:
		t.proxyNewConnectResponseUDP(serverTCP, m)
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
	for _, c := range t.configsUDP {
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
	for _, ct := range t.connectUDP {
		connects += "		" + ct.String() + ",\n"
	}
	return "	proxy[\n" +
		connects +
		"	]\n"
}
