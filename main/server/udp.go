package main

import (
	"fmt"
	"net"
	"srp-go/common/config"
	"srp-go/common/io"
	"srp-go/common/msg"
)

func startServerUDP(listenConfig *config.ListenConfig) error {
	// 启动服务器
	listenAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", listenConfig.Ip, listenConfig.Port))
	if err != nil {
		return err
	}
	listenServerUDP, err = net.ListenUDP("udp", listenAddr)
	if err != nil {
		return err
	}
	log.Info("start udp server success:", listenServerUDP.LocalAddr().String())

	// 监听请求
	go listenUDP(listenServerUDP)
	return nil
}

func listenUDP(listen *net.UDPConn) {
	udpListen := io.NewUDP(listen)
	buf := make([]byte, 64*1024)
	for {
		readLen, addr, err := udpListen.ReadFromUDP(buf)
		if err != nil {
			break
		}
		buffer := make([]byte, readLen)
		copy(buffer, buf[:readLen])
		doUDP(listen, buffer, addr)
	}
	log.Info("close udp server success:", listen.LocalAddr().String())
}

func doUDP(listen *net.UDPConn, buffer []byte, addr *net.UDPAddr) {
	udp := getUDP(addr.String(), listen)

	message, needContinue := udp.ToObj(buffer, addr, true)
	if needContinue {
		return
	}
	if message.Err != nil {
		log.Error("udp package error", message.Err)
		return
	}

	if do, err := udp.ServerInit(message.Message, addr); do {
		if err != nil {
			log.Error("udp init error", err)
		}
		return
	}

	// 交给插件处理
	for _, h := range serverHandles {
		switch m := message.Message.(type) {
		case *msg.UDPPackageConfirm:
			io.RemoteWriteUDP(m.M, m.O)
		default:
			return
		}
		if do := h.HandleUDP(udp, message.Message, addr); do {
			break
		}
	}
}

func getUDP(addr string, listen *net.UDPConn) *io.UDP {
	udpMapLock.Lock()
	defer udpMapLock.Unlock()
	udp, ok := udpMap[addr]
	if !ok {
		udp = io.NewUDP(listen)
		udpMap[addr] = udp
		go udp.GoTimeOut(func() {
			udpMapLock.Lock()
			defer udpMapLock.Unlock()
			delete(udpMap, addr)
		})
	}
	return udp
}
