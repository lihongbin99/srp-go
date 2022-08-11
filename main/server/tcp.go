package main

import (
	"fmt"
	sio "io"
	"net"
	"srp-go/common/config"
	"srp-go/common/io"
	"srp-go/common/msg"
	"srp-go/common/utils"
	"srp-go/core"
	"time"
)

// startServerTCP 启动TCP服务器
func startServerTCP(listenConfig *config.ListenConfig) error {
	// 启动服务器
	listenAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", listenConfig.Ip, listenConfig.Port))
	if err != nil {
		return err
	}
	listenServerTCP, err = net.ListenTCP("tcp", listenAddr)
	if err != nil {
		return err
	}
	log.Info("start tcp server success:", listenServerTCP.Addr().String())

	// 监听请求
	go listenTCP(listenServerTCP)
	return nil
}

// listenTCP 监听TCP请求
func listenTCP(listen *net.TCPListener) {
	for {
		conn, err := listen.AcceptTCP()
		if err != nil {
			break
		}

		// 处理请求
		go newConnectTCP(io.NewTCP(conn))
	}
	log.Info("close tcp server success:", listen.Addr().String())
}

// newConnectTCP 处理新的TCP请求
func newConnectTCP(tcp *io.TCP) {
	// 设置关流
	defer func(tcp *io.TCP) { _ = tcp.Close() }(tcp)

	// 连接初始化
	err, connectType := tcp.ServerInit()
	defer func() { utils.RemoteId(tcp.ServerId) }()
	if err != nil {
		log.Error(tcp.Base.ConnectInfo, "connect init error", err.Error())
		return
	}

	log.Info(tcp.Base.ConnectInfo, "connect", connectType)

	// 只处理 ClientTypeRegister 类型, 其他类型交给插件处理
	if connectType != msg.ClientTypeRegister {
		_ = tcp.WriteSecurityMessage(&msg.InfoResponse{Version: config.Version, ServerId: tcp.ServerId, Result: "success"})
		// 交给插件处理请求
		for _, h := range serverHandles {
			if do := h.NewConnect(tcp, connectType); do {
				log.Info(tcp.Base.ConnectInfo, "close", connectType)
				return
			}
		}
		log.Warn(tcp.Base.ConnectInfo, "no process client connect type", connectType)
		return
	}

	err = registerClient(tcp)
	log.Info(tcp.Base.ConnectInfo, "close", connectType, "returnErr", err)
}

// registerClient 处理新的注册请求
func registerClient(tcp *io.TCP) error {
	// 检测名称是否以存在
	if err := addClient(tcp); err != nil {
		_ = tcp.WriteSecurityMessage(&msg.InfoResponse{
			Version:  config.Version,
			ServerId: tcp.ServerId,
			Result:   "client name exist",
		})
		return err
	}
	defer remoteClient(tcp)
	_ = tcp.WriteSecurityMessage(&msg.InfoResponse{Version: config.Version, ServerId: tcp.ServerId, Result: "success"})

	return doRegisterClient(tcp)
}

func addClient(tcp *io.TCP) error {
	core.TcpMapLock.Lock()
	defer core.TcpMapLock.Unlock()
	if _, ok := core.TcpMap[tcp.ClientName]; ok {
		return fmt.Errorf("client name is exist")
	}
	core.TcpMap[tcp.ClientName] = tcp
	return nil
}

func remoteClient(tcp *io.TCP) {
	core.TcpMapLock.Lock()
	defer core.TcpMapLock.Unlock()
	delete(core.TcpMap, tcp.ClientName)
}

func doRegisterClient(tcp *io.TCP) error {
	tcp.Status = "Run"
	// 通知插件客户端断开连接了
	defer func() {
		for _, h := range serverHandles {
			h.Close(tcp)
		}
	}()

	// 处理读取请求
	readChan := make(chan io.Message, 8)
	go func(tcp *io.TCP, readChan chan io.Message) {
		defer close(readChan)
		for {
			message := tcp.ReadSecurityMessage(time.Time{})
			readChan <- message
			if message.Err != nil {
				break
			}
		}
	}(tcp, readChan)

	// 持续检测心跳
	keepAliveInterval := config.NewConfig.Common.KeepAlive.Interval
	keepAliveTimeout := config.NewConfig.Common.KeepAlive.Timeout
	pingTicker := time.NewTicker(time.Duration(keepAliveInterval) * time.Second)
	defer pingTicker.Stop()
	lastPingTime := time.Now()
	lastPongTime := time.Now()

	var err error = nil
	for err == nil {
		select {
		case <-pingTicker.C:
			lastPingTime = time.Now()
			log.Trace(tcp.Base.ConnectInfo, "send PingMessage")
			err = tcp.WriteSecurityMessage(&msg.PingMessage{Date: lastPingTime})
			go func() {
				time.Sleep(time.Duration(keepAliveTimeout) * time.Second)
				if lastPongTime.Before(lastPingTime) {
					log.Warn(tcp.Base.ConnectInfo, "ping timeout")
					_ = tcp.Close() // 此处直接关闭连接, 让read线程退出方法
				}
			}()
		case message := <-readChan:
			if message.Err != nil {
				err = message.Err
				break
			}
			switch m := message.Message.(type) {
			case *msg.PoneMessage:
				log.Trace(tcp.Base.ConnectInfo, "receiver Pong", m.Date)
				lastPongTime = time.Now()
			case *msg.OperateRequest:
				operateRequest(tcp, m)
			case *msg.NotifyRequest:
				notifyRequest(tcp, m)
			case *msg.NotifyResponse:
				notifyResponse(tcp, m)
			default:
				for _, h := range serverHandles {
					if do, re := h.HandleTCP(tcp, m); re {
						err = sio.EOF
						break
					} else if do {
						break
					}
				}
			}
		}
	}

	tcp.Status = "Close"
	return err
}
