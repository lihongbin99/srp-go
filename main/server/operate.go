package main

import (
	"fmt"
	"os"
	"srp-go/common/config"
	"srp-go/common/io"
	"srp-go/common/msg"
	"srp-go/core"
)

func operateRequest(tcp *io.TCP, message *msg.OperateRequest) {
	log.Info("receiver operate request", tcp.ClientName, message.Operate, message.Params)
	switch message.Operate {
	case "e":
		_ = tcp.WriteSecurityMessage(&msg.OperateResponse{
			Result: "close success",
		})
		operate(message.Operate)
	case "r":
		operate(message.Operate)
	case "c":
		operate(message.Operate)
	default:
		_ = tcp.WriteSecurityMessage(&msg.OperateResponse{
			Result: fmt.Sprintf("operate error: %s", message.Operate),
		})
	}
}

func operate(command string) {
	switch command {
	case "e":
		os.Exit(0)
	case "r":
		if err := config.RefreshConfig(); err != nil {
			log.Error("refresh config error", err)
		}
	case "c":
		connectInfo := ""
		for _, h := range serverHandles {
			connectInfo += h.ConnectInfo()
		}
		core.TcpMapLock.Lock()
		defer core.TcpMapLock.Unlock()
		clientMapInfo := ""
		for _, c := range core.TcpMap {
			clientMapInfo += "	" + c.Base.ConnectInfo.String() + ",\n"
		}
		log.Info("\n",
			"clients[\n",
			clientMapInfo,
			"]\n",
			"connectInfo[\n",
			connectInfo,
			"]\n",
		)
	default:
		log.Warn("no command", command)
	}
}
