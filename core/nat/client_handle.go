package nat

import (
	"srp-go/common/io"
	"srp-go/common/msg"
	"srp-go/common/utils"
	"sync"
)

type clientHandle struct {
	serverTCP       *io.TCP
	configs         map[string]map[string]clientConfig   // map[ServiceName]map[Protocol]clientConfig
	connectsTCP     map[utils.ID]*serverConfigConnectTCP // TCP连接对象
	connectsTCPLock sync.Mutex
	connectsUDP     map[utils.ID]*serverConfigConnectUDP // UDP连接对象
	connectsUDPLock sync.Mutex
}

func GetNewClientHandle() *clientHandle {
	return &clientHandle{
		nil,
		make(map[string]map[string]clientConfig),
		make(map[utils.ID]*serverConfigConnectTCP),
		sync.Mutex{},
		make(map[utils.ID]*serverConfigConnectUDP),
		sync.Mutex{},
	}
}

func (t *clientHandle) Start(serverTCP *io.TCP) {
	t.serverTCP = serverTCP
	// 注册服务
	go t.registerService()
}

func (t *clientHandle) Handle(serverTCP *io.TCP, message msg.Message) (do bool) {
	do = true
	switch m := message.(type) {
	case *msg.NatRegisterResponse:
		t.registerResponse(serverTCP, m)
	case *msg.NatNewConnectRequest:
		t.newConnectRequest(serverTCP, m)
	default:
		do = false
	}
	return
}

func (t *clientHandle) Close() {
	t.serverTCP = nil
}

func (t *clientHandle) ConnectInfo() string {
	connects := ""
	for _, ct := range t.connectsTCP {
		connects += "		" + ct.String() + ",\n"
	}
	for _, ct := range t.connectsUDP {
		connects += "		" + ct.String() + ",\n"
	}
	return "	nat[\n" +
		connects +
		"	]\n"
}
