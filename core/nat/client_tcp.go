package nat

import (
	"fmt"
	"net"
	"srp-go/common/config"
	"srp-go/common/io"
	"srp-go/common/msg"
	"srp-go/common/utils"
	"time"
)

func (t *clientHandle) newConnectTCP(tcp *io.TCP, c clientConfig, serviceName string, connectId utils.ID, clientAddr string) {
	log.Info(tcp.Base.ConnectInfo, "new connect", serviceName, "tcp connectId", connectId, "by", clientAddr)

	// 保存
	serverConfigTCP := &serverConfigTCP{nil, tcp.ServerId, serviceName, "tcp", c.RemotePort, c.LocalPort}
	connectTCP := &serverConfigConnectTCP{serverConfigTCP, tcp, connectId, nil, nil, clientAddr, "new"}
	t.addConnectsTCP(connectId, connectTCP)
	defer func() {
		t.connectsTCPLock.Lock()
		defer t.connectsTCPLock.Unlock()
		delete(t.connectsTCP, connectId)
	}()

	// 连接本地端口
	localAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", c.LocalPort))
	if err != nil {
		_ = tcp.WriteSecurityMessage(&msg.NatNewConnectResponse{
			ServiceName: serviceName,
			Protocol:    "tcp",
			ConnectId:   connectId,
			Result:      fmt.Sprintf("resolve tcp addr error: %s", err.Error()),
		})
		return
	}
	localConn, err := net.DialTCP("tcp", nil, localAddr)
	if err != nil {
		_ = tcp.WriteSecurityMessage(&msg.NatNewConnectResponse{
			ServiceName: serviceName,
			Protocol:    "tcp",
			ConnectId:   connectId,
			Result:      fmt.Sprintf("dial tcp addr error: %s", err.Error()),
		})
		return
	}
	defer func() { _ = localConn.Close() }()
	connectTCP.serverConnect = localConn

	// 连接远程端口
	remoteAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", config.NewConfig.Common.Server.Ip, config.NewConfig.Common.Server.Port))
	if err != nil {
		_ = tcp.WriteSecurityMessage(&msg.NatNewConnectResponse{
			ServiceName: serviceName,
			Protocol:    "tcp",
			ConnectId:   connectId,
			Result:      fmt.Sprintf("resolve tcp addr error: %s", err.Error()),
		})
		return
	}
	remoteConn, err := net.DialTCP("tcp", nil, remoteAddr)
	if err != nil {
		_ = tcp.WriteSecurityMessage(&msg.NatNewConnectResponse{
			ServiceName: serviceName,
			Protocol:    "tcp",
			ConnectId:   connectId,
			Result:      fmt.Sprintf("dial tcp addr error: %s", err.Error()),
		})
		return
	}
	defer func() { _ = remoteConn.Close() }()
	connectTCP.clientConnect = remoteConn
	remoteTCP := io.NewTCP(remoteConn)

	// 初始化
	if e := remoteTCP.ClientInit(msg.ClientTypeNat); e != nil {
		result := fmt.Sprintf("client tcp init error: %s", e.Error())
		log.Error(remoteTCP.Base.ConnectInfo, result)
		_ = tcp.WriteSecurityMessage(&msg.NatNewConnectResponse{
			ServiceName: serviceName,
			Protocol:    "tcp",
			ConnectId:   connectId,
			Result:      result,
		})
		return
	}

	// 发送成功状态
	_ = tcp.WriteSecurityMessage(&msg.NatNewConnectResponse{
		ServiceName: serviceName,
		Protocol:    "tcp",
		ConnectId:   connectId,
		Result:      "success",
	})

	// 绑定连接
	_ = remoteTCP.WriteSecurityMessage(&msg.NatAnswerConnectRequest{
		ServiceName: serviceName,
		Protocol:    "tcp",
		ConnectId:   connectId,
	})
	message := remoteTCP.ReadSecurityMessage(time.Time{})
	if message.Err != nil {
		log.Error(remoteTCP.Base.ConnectInfo, "read tcp NatAnswerConnectResponse error", message.Err)
		return
	}
	natAnswerConnectResponse, convertResult := message.Message.(*msg.NatAnswerConnectResponse)
	if !convertResult {
		log.Error(remoteTCP.Base.ConnectInfo, "read tcp NatAnswerConnectResponse type error", message.Message.GetMessageType())
		return
	}
	if natAnswerConnectResponse.Result != "success" {
		log.Error(remoteTCP.Base.ConnectInfo, "tcp NatAnswerConnectResponse result error", natAnswerConnectResponse.Result)
		return
	}

	connectTCP.status = "success"

	// 传输数据
	go remoteTCP.GoTimeOut()
	finish := make(chan interface{})
	go remoteTCP.Transfer(localConn, remoteTCP.TCPConn, finish)
	go remoteTCP.Transfer(remoteTCP.TCPConn, localConn, finish)
	_ = <-finish
	_ = <-finish
	log.Info(tcp.Base.ConnectInfo, "close", serviceName, "tcp connectId", connectId)
}

func (t *clientHandle) addConnectsTCP(connectId utils.ID, connectTCP *serverConfigConnectTCP) {
	t.connectsTCPLock.Lock()
	defer t.connectsTCPLock.Unlock()
	t.connectsTCP[connectId] = connectTCP
}
