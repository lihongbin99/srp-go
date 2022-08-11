package proxy

import (
	"fmt"
	"net"
	"srp-go/common/io"
	"srp-go/common/msg"
	"srp-go/common/utils"
	"srp-go/core"
)

func (t *serverHandle) serverProxyNewConnectRequestUDP(udp *io.UDP, clientAddr *net.UDPAddr, proxyNewConnectRequest *msg.ProxyNewConnectRequest) {
	csMainTCP, ok := core.TcpMap[proxyNewConnectRequest.TargetName]
	if !ok {
		_ = udp.WriteSecurityMessage(&msg.ProxyNewConnectResponse{
			ServerId: proxyNewConnectRequest.ServerId,
			Result:   fmt.Sprintf("no find target name by: %s", proxyNewConnectRequest.TargetName),
		}, clientAddr)
		return
	}
	proxyNewConnectRequest.ClientAddr = clientAddr.String()
	serverConnectUDP := t.addConnectUDP(proxyNewConnectRequest.ServerId, udp, proxyNewConnectRequest)
	defer t.remoteConnectUDP(proxyNewConnectRequest.ServerId)
	serverConnectUDP.ccUDP = clientAddr

	_ = csMainTCP.WriteSecurityMessage(proxyNewConnectRequest)
}

func (t *serverHandle) addConnectUDP(connectId utils.ID, udp *io.UDP, proxy *msg.ProxyNewConnectRequest) *serverConnectUDP {
	t.connectUDPLock.Lock()
	defer t.connectUDPLock.Unlock()
	serverConnectUDP := &serverConnectUDP{udp, nil, nil, connectId, proxy.ClientName, proxy.ClientAddr, proxy.TargetName, "", fmt.Sprintf("%s:%d", proxy.TargetIp, proxy.TargetPort), "new"}
	t.connectUDP[connectId] = serverConnectUDP
	return serverConnectUDP
}

func (t *serverHandle) remoteConnectUDP(connectId utils.ID) {
	t.connectUDPLock.Lock()
	defer t.connectUDPLock.Unlock()
	delete(t.connectUDP, connectId)
}

func (t *serverHandle) serverProxyNewConnectResponseUDP(_ *io.TCP, proxyNewConnectResponse *msg.ProxyNewConnectResponse) {
	ccUDPConnect, ok := t.connectUDP[proxyNewConnectResponse.ServerId]
	if ok {
		_ = ccUDPConnect.WriteSecurityMessage(proxyNewConnectResponse, ccUDPConnect.ccUDP)
		t.remoteConnectUDP(proxyNewConnectResponse.ServerId)
	} else {
		log.Error("no find ccUDP by", proxyNewConnectResponse.ServerId)
	}
}

func (t *serverHandle) udpBind(_ *io.UDP, serverAddr *net.UDPAddr, proxyNewConnectResponse *msg.ProxyNewConnectResponse) {
	ccUDPConnect, ok := t.connectUDP[proxyNewConnectResponse.ServerId]
	if !ok {
		log.Error("no find ccUDP by", proxyNewConnectResponse.ServerId)
		return
	}
	ccUDPConnect.csUDP = serverAddr
	ccUDPConnect.csAddr = serverAddr.String()

	_ = ccUDPConnect.WriteSecurityMessage(proxyNewConnectResponse, ccUDPConnect.ccUDP)
	if proxyNewConnectResponse.Result == "success" {
		t.addAddrMap(ccUDPConnect.ccUDP, ccUDPConnect.csUDP)
		return
	}
}

func (t *serverHandle) addAddrMap(ccUDP *net.UDPAddr, csUDP *net.UDPAddr) {
	// TODO 未关流
	t.addrMapLock.Lock()
	defer t.addrMapLock.Unlock()
	t.addrMap[ccUDP.String()] = csUDP
	t.addrMap[csUDP.String()] = ccUDP
}

func (t *serverHandle) serverProxyUDPPackage(udp *io.UDP, addr *net.UDPAddr, udpPackage *msg.UDPPackage) {
	t.addrMapLock.Lock()
	defer t.addrMapLock.Unlock()
	if targetAddr, ok := t.addrMap[addr.String()]; ok {
		_ = udp.WriteMessage(udpPackage, targetAddr)
		return
	}
	log.Trace("no find target addr by", addr.String())
}
