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

func (t *clientHandle) newConnectUDP(tcp *io.TCP, c clientConfig, serviceName string, connectId utils.ID, clientAddr string) {
	log.Info(tcp.Base.ConnectInfo, "new", serviceName, "udp", connectId, "by", clientAddr)

	// 保存
	serverConfigUDP := &serverConfigUDP{nil, tcp.ServerId, serviceName, "udp", c.RemotePort}
	connectUDP := &serverConfigConnectUDP{serverConfigUDP, tcp, connectId, nil, nil, nil, nil, nil, clientAddr, "new"}
	connectsUDP := t.connectsUDP
	t.addConnectsUDP(connectId, connectUDP)
	defer func() {
		t.connectsUDPLock.Lock()
		defer t.connectsUDPLock.Unlock()
		delete(connectsUDP, connectId)
	}()

	// 连接本地端口
	localAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", c.LocalPort))
	if err != nil {
		_ = tcp.WriteSecurityMessage(&msg.NatNewConnectResponse{
			ServiceName: serviceName,
			Protocol:    "udp",
			ConnectId:   connectId,
			Result:      fmt.Sprintf("resolve udp addr error: %s", err.Error()),
		})
		return
	}
	localConn, err := net.DialUDP("udp", nil, localAddr)
	if err != nil {
		_ = tcp.WriteSecurityMessage(&msg.NatNewConnectResponse{
			ServiceName: serviceName,
			Protocol:    "udp",
			ConnectId:   connectId,
			Result:      fmt.Sprintf("dial udp addr error: %s", err.Error()),
		})
		return
	}
	defer func() { _ = localConn.Close() }()

	// 连接远程端口
	remoteAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", config.NewConfig.Common.Server.Ip, config.NewConfig.Common.Server.Port))
	if err != nil {
		_ = tcp.WriteSecurityMessage(&msg.NatNewConnectResponse{
			ServiceName: serviceName,
			Protocol:    "udp",
			ConnectId:   connectId,
			Result:      fmt.Sprintf("resolve udp addr error: %s", err.Error()),
		})
		return
	}
	remoteConn, err := net.DialUDP("udp", nil, remoteAddr)
	if err != nil {
		_ = tcp.WriteSecurityMessage(&msg.NatNewConnectResponse{
			ServiceName: serviceName,
			Protocol:    "udp",
			ConnectId:   connectId,
			Result:      fmt.Sprintf("dial udp addr error: %s", err.Error()),
		})
		return
	}
	defer func() { _ = remoteConn.Close() }()
	remoteUDP := io.NewUDP(remoteConn)

	// 初始化
	if e := remoteUDP.ClientInit(msg.ClientTypeNat); e != nil {
		result := fmt.Sprintf("client udp init error: %s", e.Error())
		log.Error(remoteUDP.Base.ConnectInfo, result)
		_ = tcp.WriteSecurityMessage(&msg.NatNewConnectResponse{
			ServiceName: serviceName,
			Protocol:    "udp",
			ConnectId:   connectId,
			Result:      result,
		})
		return
	}

	// 发送成功状态
	_ = tcp.WriteSecurityMessage(&msg.NatNewConnectResponse{
		ServiceName: serviceName,
		Protocol:    "udp",
		ConnectId:   connectId,
		Result:      "success",
	})

	// 绑定连接
	_ = remoteUDP.WriteSecurityMessage(&msg.NatAnswerConnectRequest{
		ServiceName: serviceName,
		Protocol:    "udp",
		ConnectId:   connectId,
	}, nil)
	message, _, err := remoteUDP.ReadSecurityMessage(time.Time{})
	if err != nil {
		log.Error(remoteUDP.Base.ConnectInfo, "udp read NatAnswerConnectResponse error1", err)
		return
	}
	if message.Err != nil {
		log.Error(remoteUDP.Base.ConnectInfo, "udp read NatAnswerConnectResponse error2", message.Err)
		return
	}
	natAnswerConnectResponse, convertResult := message.Message.(*msg.NatAnswerConnectResponse)
	if !convertResult {
		log.Error(remoteUDP.Base.ConnectInfo, "read udp NatAnswerConnectResponse type error", message.Message.GetMessageType())
		return
	}
	if natAnswerConnectResponse.Result != "success" {
		log.Error(remoteUDP.Base.ConnectInfo, "udp NatAnswerConnectResponse result error", natAnswerConnectResponse.Result)
		return
	}

	connectUDP.status = "success"

	// 交换数据
	go remoteUDP.GoTimeOut(func() {
		_ = localConn.Close()
		_ = remoteConn.Close()
	})
	finish := make(chan interface{})
	go remoteUDP.TransferR(localConn, finish)
	go remoteUDP.TransferW(localConn, finish, msg.ClientTypeNat)
	_ = <-finish
	_ = <-finish
	log.Info(tcp.Base.ConnectInfo, "close", serviceName, "udp connectId", connectId)
}

func (t *clientHandle) addConnectsUDP(connectId utils.ID, connectUDP *serverConfigConnectUDP) {
	t.connectsUDPLock.Lock()
	defer t.connectsUDPLock.Unlock()
	t.connectsUDP[connectId] = connectUDP
}
