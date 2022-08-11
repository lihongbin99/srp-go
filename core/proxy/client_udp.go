package proxy

import (
	"fmt"
	"net"
	"srp-go/common/config"
	"srp-go/common/io"
	"srp-go/common/msg"
	"srp-go/common/utils"
	"time"
)

func (t *clientHandle) startUDP(serviceName string, clientConfig *clientConfigUDP) {
	// 监听地址
	listenAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", clientConfig.ProxyConfig.LocalPort))
	if err != nil {
		log.Error("start udp service error", serviceName, err)
		return
	}
	listen, err := net.ListenUDP("udp", listenAddr)
	if err != nil {
		log.Error("start udp service error", serviceName, err)
		return
	}

	clientConfig.listen = listen
	go t.listenUDP(serviceName, clientConfig)
}

func (t *clientHandle) listenUDP(serviceName string, clientConfig *clientConfigUDP) {
	defer func() { _ = clientConfig.listen.Close() }()
	log.Info("start udp service success", serviceName, clientConfig.listen.LocalAddr())
	buf := make([]byte, 64*1024)
	for {
		readLen, clientAddr, err := clientConfig.listen.ReadFromUDP(buf)
		if err != nil {
			break
		}
		// 处理数据
		buffer := make([]byte, readLen)
		copy(buffer, buf[:readLen])
		go t.newConnectUDP(serviceName, clientConfig, buffer, clientAddr)
	}
	log.Info("close udp service success", serviceName, clientConfig.listen.LocalAddr())
}

func (t *clientHandle) newConnectUDP(serviceName string, clientConfig *clientConfigUDP, buf []byte, clientAddr *net.UDPAddr) {
	t.connectUDPLock.Lock()
	defer t.connectUDPLock.Unlock()
	t.udpMapLock.Lock()
	defer t.udpMapLock.Unlock()
	if connectId, ok := t.udpMap[clientAddr.String()]; ok {
		if ccUDP, ok := t.connectUDP[connectId]; ok {
			if ccUDP.status != "success" {
				// 缓存数据
				cb := make([]byte, len(buf))
				copy(cb, buf)
				ccUDP.cache = append(ccUDP.cache, cb)
			} else {
				// 交换数据
				_ = ccUDP.remoteUDP.WriteMessage(&msg.UDPPackage{T: msg.ClientTypeProxy, D: buf}, nil)
			}
		}
		return
	}

	// 连接远程端口
	remoteAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", config.NewConfig.Common.Server.Ip, config.NewConfig.Common.Server.Port))
	if err != nil {
		log.Error("resolve udp addr error", serviceName, err)
		return
	}
	remoteConn, err := net.DialUDP("udp", nil, remoteAddr)
	if err != nil {
		log.Error("dial udp addr error", serviceName, err)
		return
	}
	defer func() { _ = remoteConn.Close() }()
	remoteUDP := io.NewUDP(remoteConn)

	// 初始化
	if e := remoteUDP.ClientInit(msg.ClientTypeProxy); e != nil {
		log.Error(remoteUDP.Base.ConnectInfo, "client udp init error", e)
		return
	}
	log.Info(remoteUDP.Base.ConnectInfo, "new udp proxy connect", remoteUDP.ServerId)

	// 缓存
	ccUDP := &clientConnectUDP{config.NewConfig.Common.Client.Name, clientConfig.TargetName, clientAddr.String(), fmt.Sprintf("%s:%d", clientConfig.TargetIp, clientConfig.TargetPort), remoteUDP, clientConfig.listen, "new", make([][]byte, 0)}
	// 缓存数据
	cb := make([]byte, len(buf))
	copy(cb, buf)
	ccUDP.cache = append(ccUDP.cache, cb)
	t.connectUDPLock.Lock()
	defer t.connectUDPLock.Unlock()
	t.udpMapLock.Lock()
	defer t.udpMapLock.Unlock()
	t.connectUDP[remoteUDP.ServerId] = ccUDP
	t.udpMap[clientAddr.String()] = remoteUDP.ServerId

	// TODO 此处为设置超时
	go t.createProxyUDP(remoteUDP, clientConfig, clientAddr)
}

func (t *clientHandle) createProxyUDP(remoteUDP *io.UDP, clientConfig *clientConfigUDP, clientAddr *net.UDPAddr) {
	// 尝试连接
	_ = remoteUDP.WriteSecurityMessage(&msg.ProxyNewConnectRequest{
		Protocol:   "udp",
		ClientName: config.NewConfig.Common.Client.Name,
		TargetName: clientConfig.TargetName,
		TargetIp:   clientConfig.TargetIp,
		TargetPort: clientConfig.TargetPort,
		ServerId:   remoteUDP.ServerId,
	}, nil)

	message, _, err := remoteUDP.ReadSecurityMessage(time.Time{})
	if err != nil {
		t.remoteConnectUDP(remoteUDP.ServerId)
		return
	}
	if message.Err != nil {
		log.Error(remoteUDP.Base.ConnectInfo, "read ProxyNewConnectResponse error", message.Err)
		t.remoteConnectUDP(remoteUDP.ServerId)
		return
	}
	proxyNewConnectResponse, convertResult := message.Message.(*msg.ProxyNewConnectResponse)
	if !convertResult {
		log.Error(remoteUDP.Base.ConnectInfo, "read ProxyNewConnectResponse type error", message.Message.GetMessageType())
		t.remoteConnectUDP(remoteUDP.ServerId)
		return
	}
	if proxyNewConnectResponse.Result != "success" {
		log.Error(remoteUDP.Base.ConnectInfo, "create proxy error", proxyNewConnectResponse.Result)
		t.remoteConnectUDP(remoteUDP.ServerId)
		return
	}
	t.connectUDP[remoteUDP.ServerId].status = "success"

	// 交换数据
	go remoteUDP.GoTimeOut(func() {
		_ = remoteUDP.Close()
	})
	finish := make(chan interface{})
	go remoteUDP.TransferRByUDP(clientConfig.listen, clientAddr, finish)
	_ = <-finish
}

func (t *clientHandle) proxyNewConnectResponseUDP(_ *io.TCP, proxyNewConnectResponse *msg.ProxyNewConnectResponse) {
	log.Error("create udp proxy error", proxyNewConnectResponse.ServerId, proxyNewConnectResponse.Result)
	t.remoteConnectUDP(proxyNewConnectResponse.ServerId)
}

func (t *clientHandle) proxyNewConnectRequestUDP(serverTCP *io.TCP, proxyNewConnectRequest *msg.ProxyNewConnectRequest) {
	connectUDP := &clientConnectUDP{config.NewConfig.Common.Client.Name, proxyNewConnectRequest.ClientName, fmt.Sprintf("%s:%d", proxyNewConnectRequest.TargetIp, proxyNewConnectRequest.TargetPort), proxyNewConnectRequest.ClientAddr, nil, nil, "new", nil}
	t.connectUDPLock.Lock()
	defer t.connectUDPLock.Unlock()
	t.connectUDP[proxyNewConnectRequest.ServerId] = connectUDP
	defer t.remoteConnectUDP(proxyNewConnectRequest.ServerId)

	// 连接远程端口
	remoteAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", config.NewConfig.Common.Server.Ip, config.NewConfig.Common.Server.Port))
	if err != nil {
		log.Error("resolve udp addr error", proxyNewConnectRequest.ServerId, err)
		_ = serverTCP.WriteSecurityMessage(&msg.ProxyNewConnectResponse{
			ServerId: proxyNewConnectRequest.ServerId,
			Protocol: "udp",
			Result:   fmt.Sprintf("resolve udp addr error: %s", err.Error()),
		})
		return
	}
	remoteConn, err := net.DialUDP("udp", nil, remoteAddr)
	if err != nil {
		log.Error("dial udp addr error", proxyNewConnectRequest.ServerId, err)
		_ = serverTCP.WriteSecurityMessage(&msg.ProxyNewConnectResponse{
			ServerId: proxyNewConnectRequest.ServerId,
			Protocol: "udp",
			Result:   fmt.Sprintf("dial udp addr error: %s", err.Error()),
		})
		return
	}
	defer func() { _ = remoteConn.Close() }()
	remoteUDP := io.NewUDP(remoteConn)
	connectUDP.remoteUDP = remoteUDP

	if err = remoteUDP.ClientInit(msg.ClientTypeProxy); err != nil {
		log.Error("client init error", proxyNewConnectRequest.ServerId, err)
		_ = serverTCP.WriteSecurityMessage(&msg.ProxyNewConnectResponse{
			ServerId: proxyNewConnectRequest.ServerId,
			Protocol: "udp",
			Result:   fmt.Sprintf("client init error: %s", err.Error()),
		})
		return
	}
	log.Info(remoteUDP.Base.ConnectInfo, "new udp proxy connect", proxyNewConnectRequest.TargetName, proxyNewConnectRequest.ClientAddr, "->", fmt.Sprintf("%s:%d", proxyNewConnectRequest.TargetIp, proxyNewConnectRequest.TargetPort), proxyNewConnectRequest.ServerId)

	// 连接本地端口
	localAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", proxyNewConnectRequest.TargetIp, proxyNewConnectRequest.TargetPort))
	if err != nil {
		log.Error("resolve udp addr error", proxyNewConnectRequest.ServerId, err)
		_ = remoteUDP.WriteSecurityMessage(&msg.ProxyNewConnectResponse{
			ServerId: proxyNewConnectRequest.ServerId,
			Protocol: "udp",
			Result:   fmt.Sprintf("resolve udp addr error: %s", err.Error()),
		}, nil)
		return
	}
	localConn, err := net.DialUDP("udp", nil, localAddr)
	if err != nil {
		log.Error("dial tcp addr error", proxyNewConnectRequest.ServerId, err)
		_ = remoteUDP.WriteSecurityMessage(&msg.ProxyNewConnectResponse{
			ServerId: proxyNewConnectRequest.ServerId,
			Protocol: "udp",
			Result:   fmt.Sprintf("dial udp addr error: %s", err.Error()),
		}, nil)
		return
	}
	defer func() { _ = localConn.Close() }()
	connectUDP.localUDP = localConn

	// 绑定
	_ = remoteUDP.WriteSecurityMessage(&msg.ProxyNewConnectResponse{
		ServerId: proxyNewConnectRequest.ServerId,
		Protocol: "udp",
		Result:   "success",
	}, nil)

	connectUDP.status = "success"

	// 交换数据
	go remoteUDP.GoTimeOut(func() {
		_ = localConn.Close()
		_ = remoteConn.Close()
	})
	finish := make(chan interface{})
	go remoteUDP.TransferR(localConn, finish)
	go remoteUDP.TransferW(localConn, finish, msg.ClientTypeProxy)
	_ = <-finish
	_ = <-finish
}

func (t *clientHandle) remoteConnectUDP(connectId utils.ID) {
	t.connectUDPLock.Lock()
	defer t.connectUDPLock.Unlock()
	t.udpMapLock.Lock()
	defer t.udpMapLock.Unlock()
	ccUDP, ok := t.connectUDP[connectId]
	if !ok {
		return
	}
	_ = ccUDP.remoteUDP.Close()
	delete(t.udpMap, ccUDP.localAddr)
	delete(t.connectUDP, connectId)
	log.Info("close udp proxy connect", connectId)
}
