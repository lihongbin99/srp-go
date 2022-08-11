package p2p

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
}

func GetNewServerHandle() *serverHandle {
	return &serverHandle{
		make(map[utils.ID]*serverConnectTCP),
		sync.Mutex{},
	}
}

func (t *serverHandle) NewConnect(tcp *io.TCP, connectType msg.ClientType) (do bool) {
	switch connectType {
	case msg.ClientTypeP2p:
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
	case *msg.P2pNewConnectRequest:
		t.serverP2pNewConnectRequest(tcp, m)
	case *msg.P2pNewConnectResponse:
		t.tcpBind(tcp, m)
	default:
		log.Error("read P2pNewConnectResponse type error", message.Message.GetMessageType())
	}
}

func (t *serverHandle) HandleTCP(tcp *io.TCP, message msg.Message) (do, re bool) {
	do = true
	switch m := message.(type) {
	case *msg.P2pNewConnectResponse:
		if m.Protocol == "tcp" {
			t.serverP2pNewConnectResponseTCP(tcp, m)
		} else {
			log.Error("read P2pNewConnectResponse Protocol error", m.Protocol)
		}
	default:
		do = false
	}
	return
}

func (t *serverHandle) HandleUDP(_ *io.UDP, _ msg.Message, _ *net.UDPAddr) (do bool) {
	return
}

func (t *serverHandle) Close(_ *io.TCP) {}

func (t *serverHandle) ConnectInfo() string {
	connects := t.getConnectInfoConnectsTCP()
	return "	p2p[\n" +
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
