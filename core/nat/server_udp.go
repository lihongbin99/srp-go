package nat

import (
	"fmt"
	"net"
	"srp-go/common/io"
	"srp-go/common/msg"
	"srp-go/common/utils"
)

func (t *serverHandle) registerUDP(tcp *io.TCP, message *msg.NatRegisterRequest) {
	// 监听地址
	listenAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", message.RemotePort))
	if err != nil {
		_ = tcp.WriteSecurityMessage(&msg.NatRegisterResponse{
			ServiceName: message.ServiceName,
			Protocol:    message.Protocol,
			Result:      fmt.Sprintf("resolve udp addr error: %s", err.Error()),
		})
		return
	}
	udp, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		_ = tcp.WriteSecurityMessage(&msg.NatRegisterResponse{
			ServiceName: message.ServiceName,
			Protocol:    message.Protocol,
			Result:      fmt.Sprintf("listen udp addr error: %s", err.Error()),
		})
		return
	}
	// 保存
	serverConfigUDP := &serverConfigUDP{UDPConn: udp, serviceName: message.ServiceName, protocol: message.Protocol, remotePort: message.RemotePort}
	t.addConfigsUDP(tcp, serverConfigUDP)

	_ = tcp.WriteSecurityMessage(&msg.NatRegisterResponse{
		ServiceName: message.ServiceName,
		Protocol:    message.Protocol,
		Result:      "success",
	})
	log.Info(tcp.Base.ConnectInfo, "register udp success", message.ServiceName, serverConfigUDP.LocalAddr().String())
	go t.listenUDP(tcp, serverConfigUDP)
}

func (t *serverHandle) addConfigsUDP(tcp *io.TCP, serverConfigUDP *serverConfigUDP) {
	t.configsLock.Lock()
	defer t.configsLock.Unlock()
	t.configs[tcp.ServerId] = append(t.configs[tcp.ServerId], serverConfigUDP)
}

func (t *serverHandle) listenUDP(tcp *io.TCP, serverConfigUDP *serverConfigUDP) {
	defer func() { _ = serverConfigUDP.Close() }()
	buf := make([]byte, 64*1024)
	for {
		readLen, clientAddr, err := serverConfigUDP.ReadFromUDP(buf)
		if err != nil {
			break
		}
		// 处理数据
		t.doNewUDP(tcp, serverConfigUDP, buf[:readLen], clientAddr)
		// 如果有丢包的话试一下替换下面的代码
		//buffer := make([]byte, readLen)
		//copy(buffer, buf[:readLen])
		//go t.doNewUDP(tcp, serverConfigUDP, buffer, clientAddr)
	}
	log.Info(tcp.Base.ConnectInfo, "close", serverConfigUDP.serviceName, "udp", serverConfigUDP.LocalAddr().String())
}

func (t *serverHandle) doNewUDP(tcp *io.TCP, serverConfigUDP *serverConfigUDP, buf []byte, clientAddr *net.UDPAddr) {
	t.connectsUDPLock.Lock()
	defer t.connectsUDPLock.Unlock()
	t.udpMap1Lock.Lock()
	defer t.udpMap1Lock.Unlock()
	if _, ok := t.udpMap1[serverConfigUDP.serviceName]; !ok {
		t.udpMap1[serverConfigUDP.serviceName] = make(map[string]utils.ID)
	}
	if connectId, ok := t.udpMap1[serverConfigUDP.serviceName][clientAddr.String()]; !ok {
		udp := io.NewUDP(serverConfigUDP.UDPConn)
		connectId = utils.GetId()
		sccUDP := &serverConfigConnectUDP{serverConfigUDP, tcp, connectId, clientAddr, udp, nil, nil, make([][]byte, 0), clientAddr.String(), "new"}
		// 缓存数据
		cb := make([]byte, len(buf))
		copy(cb, buf)
		sccUDP.cache = append(sccUDP.cache, cb)
		t.connectsUDP[connectId] = sccUDP
		t.udpMap1[serverConfigUDP.serviceName][clientAddr.String()] = connectId
		log.Info(tcp.Base.ConnectInfo, "new udp", serverConfigUDP.serviceName, serverConfigUDP.protocol, clientAddr, connectId)
		// TODO 此处为设置超时
		_ = tcp.WriteSecurityMessage(&msg.NatNewConnectRequest{
			ServiceName: serverConfigUDP.serviceName,
			Protocol:    "udp",
			ConnectId:   connectId,
			ClientAddr:  clientAddr.String(),
		})
	} else {
		if sccUDP, ok := t.connectsUDP[connectId]; ok {
			if sccUDP.serverConnect == nil {
				// 缓存数据
				cb := make([]byte, len(buf))
				copy(cb, buf)
				sccUDP.cache = append(sccUDP.cache, cb)
			} else if sccUDP.serverAddr != nil {
				// 交换数据
				_ = sccUDP.serverConnect.WriteMessage(&msg.UDPPackage{T: msg.ClientTypeNat, D: buf}, sccUDP.serverAddr)
			}
		}
	}
}

func (t *serverHandle) newConnectResponseUDP(tcp *io.TCP, message *msg.NatNewConnectResponse) {
	t.connectsUDPLock.Lock()
	defer t.connectsUDPLock.Unlock()
	if connect, ok := t.connectsUDP[message.ConnectId]; ok {
		if message.Result == "success" {
			connect.status = "create success"
		} else {
			delete(t.connectsUDP, message.ConnectId)
			utils.RemoteId(connect.connectId)
			log.Error(tcp.Base.ConnectInfo, "create error by", message.ServiceName, "udp", connect.clientAddr, "connectId", message.ConnectId, message.Result)
		}
	} else {
		log.Error(tcp.Base.ConnectInfo, "no find udp connect id by", message.ServiceName, message.Protocol, message.ConnectId)
	}
}

func (t *serverHandle) udpBind(serverUDP *io.UDP, message *msg.NatAnswerConnectRequest, addr *net.UDPAddr) {
	t.connectsUDPLock.Lock()
	defer t.connectsUDPLock.Unlock()
	clientUDP, ok := t.connectsUDP[message.ConnectId]
	if !ok {
		_ = serverUDP.WriteSecurityMessage(&msg.NatAnswerConnectResponse{
			ServiceName: message.ServiceName,
			Protocol:    message.Protocol,
			ConnectId:   message.ConnectId,
			Result:      fmt.Sprintf("no find udp connect id by %s %s %d", message.ServiceName, message.Protocol, message.ConnectId),
		}, addr)
		return
	}

	_ = serverUDP.WriteSecurityMessage(&msg.NatAnswerConnectResponse{
		ServiceName: message.ServiceName,
		Protocol:    message.Protocol,
		ConnectId:   message.ConnectId,
		Result:      "success",
	}, addr)

	// 之后的数据传输不在加密
	serverUDP.Security = false

	go serverUDP.GoTimeOut(func() {
		t.connectsUDPLock.Lock()
		defer t.connectsUDPLock.Unlock()
		delete(t.connectsUDP, message.ConnectId)
		delete(t.udpMap1[message.ServiceName], clientUDP.clientAddr.String())
		delete(t.udpMap2, addr.String())
		utils.RemoteId(message.ConnectId)
		log.Info(clientUDP.clientTCP.Base.ConnectInfo, "close udp", message.ServiceName, message.Protocol, clientUDP.clientAddr, message.ConnectId)
	})
	// 发出缓存数据
	for _, cb := range clientUDP.cache {
		_ = serverUDP.WriteMessage(&msg.UDPPackage{T: msg.ClientTypeNat, D: cb}, addr)
	}
	clientUDP.cache = nil

	clientUDP.serverAddr = addr
	clientUDP.serverConnect = serverUDP
	t.udpMap2[addr.String()] = message.ConnectId
}

func (t *serverHandle) transferToClient(_ *io.UDP, message *msg.UDPPackage, addr *net.UDPAddr) {
	t.connectsUDPLock.Lock()
	defer t.connectsUDPLock.Unlock()
	if connectId, ok := t.udpMap2[addr.String()]; ok {
		if clientUDP, ok := t.connectsUDP[connectId]; ok {
			if clientUDP.clientConnect != nil && clientUDP.clientAddr != nil {
				_, _ = clientUDP.clientConnect.WriteToUDP(message.D, clientUDP.clientAddr)
			}
		}
	}
}
