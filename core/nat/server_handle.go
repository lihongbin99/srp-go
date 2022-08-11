package nat

import (
	"net"
	"srp-go/common/io"
	"srp-go/common/msg"
	"srp-go/common/utils"
	"strconv"
	"sync"
)

type serverHandle struct {
	configs         map[utils.ID][]serverConfig // 服务器配置 map[ServerId][]serverConfig
	configsLock     sync.Mutex
	connectsTCP     map[utils.ID]*serverConfigConnectTCP // UDP连接对象
	connectsTCPLock sync.Mutex
	connectsUDP     map[utils.ID]*serverConfigConnectUDP // UDP连接对象
	connectsUDPLock sync.Mutex
	udpMap1         map[string]map[string]utils.ID // UDP map[ServiceName][Addr]ConnectId
	udpMap1Lock     sync.Mutex
	udpMap2         map[string]utils.ID // UDP map[Addr]ConnectId
	udpMap2Lock     sync.Mutex
}

func GetNewServerHandle() *serverHandle {
	return &serverHandle{
		make(map[utils.ID][]serverConfig),
		sync.Mutex{},
		make(map[utils.ID]*serverConfigConnectTCP),
		sync.Mutex{},
		make(map[utils.ID]*serverConfigConnectUDP),
		sync.Mutex{},
		make(map[string]map[string]utils.ID),
		sync.Mutex{},
		make(map[string]utils.ID),
		sync.Mutex{},
	}
}

func (t *serverHandle) NewConnect(tcp *io.TCP, connectType msg.ClientType) (do bool) {
	switch connectType {
	case msg.ClientTypeNat:
		t.tcpBind(tcp)
		return true
	}
	return
}

func (t *serverHandle) HandleTCP(tcp *io.TCP, message msg.Message) (do, re bool) {
	do = true
	switch m := message.(type) {
	case *msg.NatRegisterRequest:
		t.register(tcp, m)
	case *msg.NatNewConnectResponse:
		t.newConnectResponse(tcp, m)
	default:
		do = false
	}
	return
}

func (t *serverHandle) HandleUDP(udp *io.UDP, message msg.Message, addr *net.UDPAddr) (do bool) {
	do = true
	switch m := message.(type) {
	case *msg.NatAnswerConnectRequest:
		t.udpBind(udp, m, addr)
	case *msg.UDPPackage:
		if m.T == msg.ClientTypeNat {
			t.transferToClient(udp, m, addr)
		} else {
			do = false
		}
	default:
		do = false
	}
	return
}

func (t *serverHandle) Close(tcp *io.TCP) {
	t.configsLock.Lock()
	defer t.configsLock.Unlock()
	// 关闭监听的端口
	if configs, ok := t.configs[tcp.ServerId]; ok {
		for _, c := range configs {
			c.closeConfig()
		}
		delete(t.configs, tcp.ServerId)
	}
}

func (t *serverHandle) ConnectInfo() string {
	listenS := t.getConnectInfoConfigs()
	connects := t.getConnectInfoConnectsTCP() + t.getConnectInfoConnectsUDP()
	return "	nat[\n" +
		"		listen[\n" +
		listenS +
		"		]\n" +
		"		connects[\n" +
		connects +
		"		]\n" +
		"	]\n"
}

func (t *serverHandle) getConnectInfoConfigs() (listenS string) {
	t.configsLock.Lock()
	defer t.configsLock.Unlock()
	for serverId, cs := range t.configs {
		for _, c := range cs {
			listenS += "			" + strconv.FormatUint(serverId, 10) + "-" + c.String() + ",\n"
		}
	}
	return
}

func (t *serverHandle) getConnectInfoConnectsTCP() (connects string) {
	t.connectsTCPLock.Lock()
	defer t.connectsTCPLock.Unlock()
	for _, ct := range t.connectsTCP {
		connects += "			" + ct.String() + ",\n"
	}
	return
}

func (t *serverHandle) getConnectInfoConnectsUDP() (connects string) {
	t.connectsUDPLock.Lock()
	defer t.connectsUDPLock.Unlock()
	for _, ct := range t.connectsUDP {
		connects += "			" + ct.String() + ",\n"
	}
	return
}
