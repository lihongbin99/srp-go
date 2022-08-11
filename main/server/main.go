package main

import (
	"net"
	"os"
	"srp-go/common/config"
	"srp-go/common/io"
	"srp-go/common/logger"
	"srp-go/core/nat"
	"srp-go/core/p2p"
	"srp-go/core/proxy"
	"strings"
	"sync"
	"time"
)

// config\s.ini
var (
	log = logger.NewLog("Server Main")

	listenServerTCP *net.TCPListener = nil
	listenServerUDP *net.UDPConn     = nil

	udpMap     = make(map[string]*io.UDP)
	udpMapLock = sync.Mutex{}

	serverHandles = make(map[string]ServerHandle)
)

func init() {
	serverHandles["nat"] = nat.GetNewServerHandle()
	serverHandles["proxy"] = proxy.GetNewServerHandle()
	serverHandles["p2p"] = p2p.GetNewServerHandle()

	config.AddFlushConfigHandle(flushConfig)
}

func main() {
	// 启动服务器
	if err := startServer(); err != nil {
		log.Error("start server error", err)
	}
	// 处理命令行
	buf := make([]byte, 64*1024)
	for {
		readLen, err := os.Stdin.Read(buf)
		if err != nil {
			log.Warn("read form console error", err)
			break
		}
		operate(strings.TrimSpace(string(buf[:readLen])))
	}
	for {
		time.Sleep(1 * time.Hour)
	}
}

func flushConfig(oldConfig *config.Config, newConfig *config.Config) {
	if oldConfig.Common.Listen.Ip != newConfig.Common.Listen.Ip ||
		oldConfig.Common.Listen.Port != newConfig.Common.Listen.Port {
		if err := restartServer(); err != nil {
			log.Error("restart server error", err)
		} else {
			log.Info("restart server success, new server", listenServerTCP.Addr())
		}
	}
}

func restartServer() error {
	closeServer()
	return startServer()
}

func closeServer() {
	if listenServerTCP != nil {
		_ = listenServerTCP.Close()
		listenServerTCP = nil
	}
	if listenServerUDP != nil {
		_ = listenServerUDP.Close()
		listenServerUDP = nil
	}
}

func startServer() error {
	log.Info("start server")
	if config.NewConfig.Common.Listen.Port <= 0 {
		log.Debug("port < 0, no start server")
		return nil
	}
	if err := startServerTCP(config.NewConfig.Common.Listen); err != nil {
		return err
	}
	if err := startServerUDP(config.NewConfig.Common.Listen); err != nil {
		return err
	}
	return nil
}
