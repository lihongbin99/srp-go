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

func (t *clientHandle) startTCP(serviceName string, clientConfig *clientConfigTCP) {
	// 监听地址
	listenAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", clientConfig.ProxyConfig.LocalPort))
	if err != nil {
		log.Error("start tcp service error", serviceName, err)
		return
	}
	listen, err := net.ListenTCP("tcp", listenAddr)
	if err != nil {
		log.Error("start tcp service error", serviceName, err)
		return
	}

	clientConfig.listen = listen
	go t.listenTCP(serviceName, clientConfig)
}

func (t *clientHandle) listenTCP(serviceName string, clientConfig *clientConfigTCP) {
	defer func() { _ = clientConfig.listen.Close() }()
	log.Info("start tcp service success", serviceName, clientConfig.listen.Addr())
	for {
		conn, err := clientConfig.listen.AcceptTCP()
		if err != nil {
			break
		}
		go t.newConnectTCP(serviceName, conn, clientConfig)
	}
	log.Info("close tcp service success", serviceName, clientConfig.listen.Addr())
}

func (t *clientHandle) newConnectTCP(serviceName string, localConn *net.TCPConn, clientConfig *clientConfigTCP) {
	defer func() { _ = localConn.Close() }()

	// 连接远程端口
	remoteAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", config.NewConfig.Common.Server.Ip, config.NewConfig.Common.Server.Port))
	if err != nil {
		log.Error("resolve tcp addr error", serviceName, err)
		return
	}
	remoteConn, err := net.DialTCP("tcp", nil, remoteAddr)
	if err != nil {
		log.Error("dial tcp addr error", serviceName, err)
		return
	}
	defer func() { _ = remoteConn.Close() }()
	remoteTCP := io.NewTCP(remoteConn)

	// 初始化
	if e := remoteTCP.ClientInit(msg.ClientTypeProxy); e != nil {
		log.Error(remoteTCP.Base.ConnectInfo, "client tcp init error", e)
		return
	}
	// 保存起来查询用
	connectTCP := t.addConnectTCP(remoteTCP.ServerId, config.NewConfig.Common.Client.Name, clientConfig.TargetName, localConn.LocalAddr().String(), fmt.Sprintf("%s:%d", clientConfig.TargetIp, clientConfig.TargetPort))
	defer t.remoteConnectTCP(remoteTCP.ServerId)
	connectTCP.localTCP = localConn
	connectTCP.remoteTCP = remoteTCP
	log.Info(remoteTCP.Base.ConnectInfo, "new tcp proxy connect", remoteTCP.ServerId)

	// 尝试连接
	_ = remoteTCP.WriteSecurityMessage(&msg.ProxyNewConnectRequest{
		Protocol:   "tcp",
		TargetName: clientConfig.TargetName,
		TargetIp:   clientConfig.TargetIp,
		TargetPort: clientConfig.TargetPort,
		ServerId:   remoteTCP.ServerId,
	})

	message := remoteTCP.ReadSecurityMessage(time.Time{})
	if message.Err != nil {
		log.Error(remoteTCP.Base.ConnectInfo, "read ProxyNewConnectResponse error", message.Err)
		return
	}
	proxyNewConnectResponse, convertResult := message.Message.(*msg.ProxyNewConnectResponse)
	if !convertResult {
		log.Error(remoteTCP.Base.ConnectInfo, "read ProxyNewConnectResponse type error", message.Message.GetMessageType())
		return
	}
	if proxyNewConnectResponse.Result != "success" {
		log.Error(remoteTCP.Base.ConnectInfo, "create proxy error", proxyNewConnectResponse.Result)
		return
	}
	connectTCP.status = "success"

	// 开始传输数据
	transfer(remoteTCP, localConn, remoteTCP.ServerId)
}

func (t *clientHandle) proxyNewConnectRequestTCP(serverTCP *io.TCP, proxyNewConnectRequest *msg.ProxyNewConnectRequest) {
	connectTCP := t.addConnectTCP(proxyNewConnectRequest.ServerId, config.NewConfig.Common.Client.Name, proxyNewConnectRequest.ClientName, fmt.Sprintf("%s:%d", proxyNewConnectRequest.TargetIp, proxyNewConnectRequest.TargetPort), proxyNewConnectRequest.ClientAddr)
	defer t.remoteConnectTCP(proxyNewConnectRequest.ServerId)

	// 连接远程端口
	remoteAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", config.NewConfig.Common.Server.Ip, config.NewConfig.Common.Server.Port))
	if err != nil {
		log.Error("resolve tcp addr error", proxyNewConnectRequest.ServerId, err)
		_ = serverTCP.WriteSecurityMessage(&msg.ProxyNewConnectResponse{
			ServerId: proxyNewConnectRequest.ServerId,
			Protocol: "tcp",
			Result:   fmt.Sprintf("resolve tcp addr error: %s", err.Error()),
		})
		return
	}
	remoteConn, err := net.DialTCP("tcp", nil, remoteAddr)
	if err != nil {
		log.Error("dial tcp addr error", proxyNewConnectRequest.ServerId, err)
		_ = serverTCP.WriteSecurityMessage(&msg.ProxyNewConnectResponse{
			ServerId: proxyNewConnectRequest.ServerId,
			Protocol: "tcp",
			Result:   fmt.Sprintf("dial tcp addr error: %s", err.Error()),
		})
		return
	}
	defer func() { _ = remoteConn.Close() }()
	remoteTCP := io.NewTCP(remoteConn)
	connectTCP.remoteTCP = remoteTCP

	if err = remoteTCP.ClientInit(msg.ClientTypeProxy); err != nil {
		log.Error("client init error", proxyNewConnectRequest.ServerId, err)
		_ = serverTCP.WriteSecurityMessage(&msg.ProxyNewConnectResponse{
			ServerId: proxyNewConnectRequest.ServerId,
			Protocol: "tcp",
			Result:   fmt.Sprintf("client init error: %s", err.Error()),
		})
		return
	}
	log.Info(remoteTCP.Base.ConnectInfo, "new proxy connect", proxyNewConnectRequest.TargetName, proxyNewConnectRequest.ClientAddr, "->", fmt.Sprintf("%s:%d", proxyNewConnectRequest.TargetIp, proxyNewConnectRequest.TargetPort), proxyNewConnectRequest.ServerId)

	// 连接本地端口
	localAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", proxyNewConnectRequest.TargetIp, proxyNewConnectRequest.TargetPort))
	if err != nil {
		log.Error("resolve tcp addr error", proxyNewConnectRequest.ServerId, err)
		_ = remoteTCP.WriteSecurityMessage(&msg.ProxyNewConnectResponse{
			ServerId: proxyNewConnectRequest.ServerId,
			Protocol: "tcp",
			Result:   fmt.Sprintf("resolve tcp addr error: %s", err.Error()),
		})
		return
	}
	localConn, err := net.DialTCP("tcp", nil, localAddr)
	if err != nil {
		log.Error("dial tcp addr error", proxyNewConnectRequest.ServerId, err)
		_ = remoteTCP.WriteSecurityMessage(&msg.ProxyNewConnectResponse{
			ServerId: proxyNewConnectRequest.ServerId,
			Protocol: "tcp",
			Result:   fmt.Sprintf("dial tcp addr error: %s", err.Error()),
		})
		return
	}
	defer func() { _ = localConn.Close() }()
	connectTCP.localTCP = localConn

	// 绑定
	_ = remoteTCP.WriteSecurityMessage(&msg.ProxyNewConnectResponse{
		ServerId: proxyNewConnectRequest.ServerId,
		Protocol: "tcp",
		Result:   "success",
	})

	connectTCP.status = "success"

	// 交换数据
	transfer(remoteTCP, localConn, proxyNewConnectRequest.ServerId)
}

func transfer(remoteTCP *io.TCP, localConn *net.TCPConn, connectId utils.ID) {
	go remoteTCP.GoTimeOut()
	finish := make(chan interface{})
	go remoteTCP.Transfer(localConn, remoteTCP.TCPConn, finish)
	go remoteTCP.Transfer(remoteTCP.TCPConn, localConn, finish)
	_ = <-finish
	_ = <-finish
	log.Info(remoteTCP.Base.ConnectInfo, "close proxy connect", connectId)
}

func (t *clientHandle) addConnectTCP(connectId utils.ID, thisName string, targetName string, localAddr string, targetAddr string) *clientConnectTCP {
	t.connectTCPLock.Lock()
	defer t.connectTCPLock.Unlock()
	connectTCP := &clientConnectTCP{thisName, targetName, localAddr, targetAddr, nil, nil, "new"}
	t.connectTCP[connectId] = connectTCP
	return connectTCP
}

func (t *clientHandle) remoteConnectTCP(connectId utils.ID) {
	t.connectTCPLock.Lock()
	defer t.connectTCPLock.Unlock()
	delete(t.connectTCP, connectId)
}
