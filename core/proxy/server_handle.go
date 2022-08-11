package proxy

import (
	"net"
	"srp-go/common/io"
	"srp-go/common/msg"
	"srp-go/common/utils"
	"sync"
	"time"
)

type serverHandle struct {
	connectTCP     map[utils.ID]*serverConnectTCP
	connectTCPLock sync.Mutex
	connectUDP     map[utils.ID]*serverConnectUDP
	connectUDPLock sync.Mutex
	addrMap        map[string]*net.UDPAddr
	addrMapLock    sync.Mutex
}

func GetNewServerHandle() *serverHandle {
	return &serverHandle{
		make(map[utils.ID]*serverConnectTCP),
		sync.Mutex{},
		make(map[utils.ID]*serverConnectUDP),
		sync.Mutex{},
		make(map[string]*net.UDPAddr),
		sync.Mutex{},
	}
}

func (t *serverHandle) NewConnect(tcp *io.TCP, connectType msg.ClientType) (do bool) {
	switch connectType {
	case msg.ClientTypeProxy:
		t.newConnect(tcp)
		return true
	}
	return
}

func (t *serverHandle) newConnect(tcp *io.TCP) {
	message := tcp.ReadSecurityMessage(time.Time{})
	if message.Err != nil {
		log.Error("read fast message error", message.Err)
		return
	}
	switch m := message.Message.(type) {
	case *msg.ProxyNewConnectRequest:
		t.serverProxyNewConnectRequest(tcp, m)
	case *msg.ProxyNewConnectResponse:
		t.tcpBind(tcp, m)
	default:
		log.Error("read ProxyNewConnectResponse type error", message.Message.GetMessageType())
	}
}

func (t *serverHandle) HandleTCP(tcp *io.TCP, message msg.Message) (do, re bool) {
	do = true
	switch m := message.(type) {
	case *msg.ProxyNewConnectResponse:
		if m.Protocol == "tcp" {
			t.serverProxyNewConnectResponseTCP(tcp, m)
		} else if m.Protocol == "udp" {
			t.serverProxyNewConnectResponseUDP(tcp, m)
		} else {
			log.Error("read ProxyNewConnectResponse Protocol error", m.Protocol)
		}
	default:
		do = false
	}
	return
}

func (t *serverHandle) HandleUDP(udp *io.UDP, message msg.Message, addr *net.UDPAddr) (do bool) {
	do = true
	switch m := message.(type) {
	case *msg.ProxyNewConnectRequest:
		t.serverProxyNewConnectRequestUDP(udp, addr, m)
	case *msg.ProxyNewConnectResponse:
		t.udpBind(udp, addr, m)
	case *msg.UDPPackage:
		if m.T == msg.ClientTypeProxy {
			t.serverProxyUDPPackage(udp, addr, m)
		} else {
			do = false
		}
	default:
		do = false
	}
	return
}

func (t *serverHandle) Close(_ *io.TCP) {}

func (t *serverHandle) ConnectInfo() string {
	connects := t.getConnectInfoConnectsTCP() + t.getConnectInfoConnectsUDP()
	return "	proxy[\n" +
		connects +
		"	]\n"
}

func (t *serverHandle) getConnectInfoConnectsTCP() (connects string) {
	t.connectTCPLock.Lock()
	defer t.connectTCPLock.Unlock()
	for _, ct := range t.connectTCP {
		connects += "			" + ct.String() + ",\n"
	}
	return
}

func (t *serverHandle) getConnectInfoConnectsUDP() (connects string) {
	t.connectUDPLock.Lock()
	defer t.connectUDPLock.Unlock()
	for _, ct := range t.connectUDP {
		connects += "			" + ct.String() + ",\n"
	}
	return
}
