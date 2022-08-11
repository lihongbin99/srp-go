package nat

import (
	"fmt"
	"net"
	"srp-go/common/io"
	"srp-go/common/msg"
	"srp-go/common/utils"
	"time"
)

func (t *serverHandle) registerTCP(tcp *io.TCP, message *msg.NatRegisterRequest) {
	// 监听地址
	listenAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", message.RemotePort))
	if err != nil {
		_ = tcp.WriteSecurityMessage(&msg.NatRegisterResponse{
			ServiceName: message.ServiceName,
			Protocol:    message.Protocol,
			Result:      fmt.Sprintf("resolve tcp addr error: %s", err.Error()),
		})
		return
	}
	listen, err := net.ListenTCP("tcp", listenAddr)
	if err != nil {
		_ = tcp.WriteSecurityMessage(&msg.NatRegisterResponse{
			ServiceName: message.ServiceName,
			Protocol:    message.Protocol,
			Result:      fmt.Sprintf("listen tcp addr error: %s", err.Error()),
		})
		return
	}

	// 保存
	serverConfigTCP := &serverConfigTCP{TCPListener: listen, serverId: tcp.ServerId, serviceName: message.ServiceName, protocol: message.Protocol, remotePort: message.RemotePort, localPort: message.LocalPort}
	t.addConfigsTCP(tcp, serverConfigTCP)

	_ = tcp.WriteSecurityMessage(&msg.NatRegisterResponse{
		ServiceName: message.ServiceName,
		Protocol:    message.Protocol,
		Result:      "success",
	})
	log.Info(tcp.Base.ConnectInfo, "register tcp success", message.ServiceName, "tcp", serverConfigTCP.Addr().String())
	go t.listenTCP(tcp, serverConfigTCP)
}

func (t *serverHandle) addConfigsTCP(tcp *io.TCP, serverConfigTCP *serverConfigTCP) {
	t.configsLock.Lock()
	defer t.configsLock.Unlock()
	t.configs[tcp.ServerId] = append(t.configs[tcp.ServerId], serverConfigTCP)
}

func (t *serverHandle) listenTCP(tcp *io.TCP, serverConfigTCP *serverConfigTCP) {
	defer func() { _ = serverConfigTCP.Close() }()
	for {
		conn, err := serverConfigTCP.AcceptTCP()
		if err != nil {
			break
		}
		// 保存
		connectId := utils.GetId()
		serverConfigConnectTCP := &serverConfigConnectTCP{serverConfigTCP, tcp, connectId, conn, nil, conn.RemoteAddr().String(), "new"}
		t.addServerConfigConnectTCP(connectId, serverConfigConnectTCP)
		log.Info(tcp.Base.ConnectInfo, "new tcp", serverConfigTCP.serviceName, "tcp", conn.RemoteAddr(), "connectId", connectId)

		_ = tcp.WriteSecurityMessage(&msg.NatNewConnectRequest{
			ServiceName: serverConfigTCP.serviceName,
			Protocol:    "tcp",
			ConnectId:   connectId,
			ClientAddr:  conn.RemoteAddr().String(),
		})
	}
	log.Info(tcp.Base.ConnectInfo, "close", serverConfigTCP.serviceName, "tcp", serverConfigTCP.Addr().String())
}

func (t *serverHandle) addServerConfigConnectTCP(connectId utils.ID, serverConfigConnectTCP *serverConfigConnectTCP) {
	t.connectsTCPLock.Lock()
	defer t.connectsTCPLock.Unlock()
	t.connectsTCP[connectId] = serverConfigConnectTCP
}

func (t *serverHandle) newConnectResponseTCP(tcp *io.TCP, message *msg.NatNewConnectResponse) {
	t.connectsTCPLock.Lock()
	defer t.connectsTCPLock.Unlock()
	if connect, ok := t.connectsTCP[message.ConnectId]; ok {
		if message.Result == "success" {
			connect.status = "success"
		} else {
			connect.Close()
			delete(t.connectsTCP, message.ConnectId)
			log.Info(tcp.Base.ConnectInfo, "create error by", message.ServiceName, "tcp", connect.clientConnect.RemoteAddr(), "connectId", message.ConnectId, message.Result)
		}
	} else {
		log.Error(tcp.Base.ConnectInfo, "connect response not find connectId by", message.ServiceName, "tcp", "connectId", message.ConnectId)
	}
}

func (t *serverHandle) tcpBind(serverTCP *io.TCP) {
	message := serverTCP.ReadSecurityMessage(time.Time{})
	if message.Err != nil {
		log.Error(serverTCP.Base.ConnectInfo, "read NatAnswerConnectRequest error", message.Err)
		return
	}
	natAnswerConnectRequest, convertResult := message.Message.(*msg.NatAnswerConnectRequest)
	if !convertResult {
		log.Error(serverTCP.Base.ConnectInfo, "read NatAnswerConnectRequest type error", message.Message.GetMessageType())
		return
	}

	clientTCP, ok := t.connectsTCP[natAnswerConnectRequest.ConnectId]
	if !ok {
		_ = serverTCP.WriteSecurityMessage(&msg.NatAnswerConnectResponse{
			ServiceName: natAnswerConnectRequest.ServiceName,
			Protocol:    natAnswerConnectRequest.Protocol,
			ConnectId:   natAnswerConnectRequest.ConnectId,
			Result:      fmt.Sprintf("tcp bind not find tcp connect id by %s %s %d", natAnswerConnectRequest.ServiceName, natAnswerConnectRequest.Protocol, natAnswerConnectRequest.ConnectId),
		})
		return
	}

	defer func() {
		t.connectsTCPLock.Lock()
		defer t.connectsTCPLock.Unlock()
		clientTCP.Close()
		delete(t.connectsTCP, natAnswerConnectRequest.ConnectId)
		log.Info(clientTCP.clientTCP.Base.ConnectInfo, "close tcp", natAnswerConnectRequest.ServiceName, natAnswerConnectRequest.Protocol, clientTCP.clientConnect.RemoteAddr(), natAnswerConnectRequest.ConnectId)
	}()

	clientTCP.serverConnect = serverTCP.TCPConn

	_ = serverTCP.WriteSecurityMessage(&msg.NatAnswerConnectResponse{
		ServiceName: natAnswerConnectRequest.ServiceName,
		Protocol:    natAnswerConnectRequest.Protocol,
		ConnectId:   natAnswerConnectRequest.ConnectId,
		Result:      "success",
	})

	// 交换数据
	go serverTCP.GoTimeOut()
	finish := make(chan interface{})
	go serverTCP.Transfer(clientTCP.clientConnect, serverTCP.TCPConn, finish)
	go serverTCP.Transfer(serverTCP.TCPConn, clientTCP.clientConnect, finish)
	_ = <-finish
	_ = <-finish
}
