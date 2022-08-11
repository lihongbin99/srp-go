package main

import (
	"fmt"
	"net"
	"os"
	"srp-go/common/config"
	"srp-go/common/io"
	"srp-go/common/logger"
	"srp-go/common/msg"
	"srp-go/core/nat"
	"srp-go/core/p2p"
	"srp-go/core/proxy"
	"strings"
	"time"
)

// config\c-c.ini
// config\c-s.ini
var (
	log = logger.NewLog("Main")

	serverTCP *io.TCP = nil

	clientHandles = make([]ClientHandle, 0)
)

func init() {
	clientHandles = append(clientHandles, nat.GetNewClientHandle())
	clientHandles = append(clientHandles, proxy.GetNewClientHandle())
	clientHandles = append(clientHandles, p2p.GetNewClientHandle())
}

func main() {
	// 启动服务
	startServer()
	// 处理命令行
	buf := make([]byte, 64*1024)
	for {
		readLen, err := os.Stdin.Read(buf)
		if err != nil {
			log.Error("read form console error", err)
			break
		}
		doOperate(strings.TrimSpace(string(buf[:readLen])))
	}
}

func closeServer() {
	if serverTCP != nil {
		_ = serverTCP.Close()
		serverTCP = nil
	}
}

// startServer 启动服务, 如果断开了就不断重试
func startServer() {
	go func() {
		interval := 1
		for {
			success := startService()
			if success {
				interval = 1
			}
			time.Sleep(time.Duration(interval) * time.Second)
			interval = interval * 2
			if interval > 60 {
				interval = 60
			}
		}
	}()
}

// startService 初始化连接
func startService() (success bool) {
	if config.NewConfig.Common.Server.Port <= 0 {
		return false
	}
	serverAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", config.NewConfig.Common.Server.Ip, config.NewConfig.Common.Server.Port))
	if err != nil {
		log.Error("resolve server addr error", serverAddr)
		return
	}
	serverConn, err := net.DialTCP("tcp", nil, serverAddr)
	if err != nil {
		log.Error("dial server error", err)
		return
	}

	tcp := io.NewTCP(serverConn)
	serverTCP = tcp
	defer func(tcp *io.TCP) {
		_ = tcp.Close()
		for _, h := range clientHandles {
			h.Close()
		}
	}(tcp)

	// 初始化连接
	if e := tcp.ClientInit(msg.ClientTypeRegister); e != nil {
		log.Error(tcp.Base.ConnectInfo, "client init error", e.Error())
		return
	}

	log.Info(tcp.Base.ConnectInfo, "connect server success")

	err = doStartService(tcp)
	log.Info(tcp.Base.ConnectInfo, "close server returnErr", err)
	return true
}

// startService 启动服务
func doStartService(tcp *io.TCP) error {
	tcp.Status = "Run"

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

	// 通知插件
	for _, h := range clientHandles {
		h.Start(tcp)
	}

	// 持续检测心跳
	keepAliveInterval := config.NewConfig.Common.KeepAlive.Interval
	keepAliveTimeout := config.NewConfig.Common.KeepAlive.Timeout
	pingTicker := time.NewTicker(time.Duration(keepAliveInterval+keepAliveTimeout) * time.Second)
	defer pingTicker.Stop()
	lastPongTime := time.Now()
	lastPingTime := time.Now()
	var err error = nil
	for err == nil {
		select {
		case <-pingTicker.C:
			if lastPongTime.Before(lastPingTime) {
				err = fmt.Errorf("ping timeout")
			}
			lastPingTime = time.Now()
		case message := <-readChan:
			if message.Err != nil {
				err = message.Err
				break
			}
			switch m := message.Message.(type) {
			case *msg.PingMessage:
				lastPongTime = time.Now()
				log.Trace(tcp.Base.ConnectInfo, "receiver PingMessage", m.Date)
				err = tcp.WriteSecurityMessage(&msg.PoneMessage{Date: time.Now()})
			case *msg.OperateResponse:
				log.Info("operate result", m.Result)
			case *msg.NotifyRequest:
				doNotify(tcp, m)
			case *msg.NotifyResponse:
				log.Info("notify result", m.Result)
			default:
				for _, h := range clientHandles {
					if do := h.Handle(tcp, m); do {
						break
					}
				}
			}
		}
	}

	tcp.Status = "Close"
	return err
}
