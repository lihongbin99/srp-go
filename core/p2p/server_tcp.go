package p2p

import (
	"fmt"
	"srp-go/common/io"
	"srp-go/common/msg"
	"srp-go/common/utils"
	"srp-go/core"
)

func (t *serverHandle) serverP2pNewConnectRequest(ccTCP *io.TCP, proxyNewConnectRequest *msg.P2pNewConnectRequest) {
	csMainTCP, ok := core.TcpMap[proxyNewConnectRequest.TargetName]
	if !ok {
		_ = ccTCP.WriteSecurityMessage(&msg.ProxyNewConnectResponse{
			ServerId: proxyNewConnectRequest.ServerId,
			Result:   fmt.Sprintf("no find target name by: %s", proxyNewConnectRequest.TargetName),
		})
		return
	}
	proxyNewConnectRequest.ClientAddr = ccTCP.RemoteAddr().String()
	proxyNewConnectRequest.ClientName = ccTCP.ClientName
	serverConnectTCP := t.addConnectTCP(proxyNewConnectRequest.ServerId, proxyNewConnectRequest)
	defer t.remoteConnectTCP(proxyNewConnectRequest.ServerId)
	serverConnectTCP.ccTCP = ccTCP

	_ = csMainTCP.WriteSecurityMessage(proxyNewConnectRequest)

	// 等待
	if r := <-serverConnectTCP.waitChan; !r {
		return
	}
	serverConnectTCP.status = "success"

	// 交换数据
	csTCP := serverConnectTCP.csTCP
	go ccTCP.GoTimeOut()
	finish := make(chan interface{})
	go ccTCP.Transfer(ccTCP.TCPConn, csTCP.TCPConn, finish)
	_ = <-finish
}

func (t *serverHandle) addConnectTCP(connectId utils.ID, proxy *msg.P2pNewConnectRequest) *serverConnectTCP {
	t.connectTCPLock.Lock()
	defer t.connectTCPLock.Unlock()
	serverConnectTCP := &serverConnectTCP{nil, nil, make(chan bool), connectId, proxy.ClientName, proxy.ClientAddr, proxy.TargetName, "", fmt.Sprintf("%s:%d", proxy.TargetIp, proxy.TargetPort), "new"}
	t.connectTCP[connectId] = serverConnectTCP
	return serverConnectTCP
}

func (t *serverHandle) remoteConnectTCP(connectId utils.ID) {
	t.connectTCPLock.Lock()
	defer t.connectTCPLock.Unlock()
	delete(t.connectTCP, connectId)
}

func (t *serverHandle) serverP2pNewConnectResponseTCP(_ *io.TCP, proxyNewConnectResponse *msg.P2pNewConnectResponse) {
	ccTCPConnect, ok := t.connectTCP[proxyNewConnectResponse.ServerId]
	if ok {
		_ = ccTCPConnect.ccTCP.WriteSecurityMessage(proxyNewConnectResponse)
		ccTCPConnect.waitChan <- false
	} else {
		log.Error("no find ccTCP by", proxyNewConnectResponse.ServerId)
	}
}

func (t *serverHandle) tcpBind(csTCP *io.TCP, proxyNewConnectResponse *msg.P2pNewConnectResponse) {
	ccTCPConnect, ok := t.connectTCP[proxyNewConnectResponse.ServerId]
	if !ok {
		log.Error("no find ccTCP by", proxyNewConnectResponse.ServerId)
		return
	}
	ccTCPConnect.csTCP = csTCP
	ccTCPConnect.csAddr = csTCP.RemoteAddr().String()

	_ = ccTCPConnect.ccTCP.WriteSecurityMessage(proxyNewConnectResponse)
	if proxyNewConnectResponse.Result != "success" {
		ccTCPConnect.waitChan <- false
		return
	}
	ccTCPConnect.waitChan <- true

	// 交换数据
	ccTCP := ccTCPConnect.ccTCP
	go csTCP.GoTimeOut()
	finish := make(chan interface{})
	go csTCP.Transfer(csTCP.TCPConn, ccTCP.TCPConn, finish)
	_ = <-finish
}
