package main

import (
	"fmt"
	"os"
	"srp-go/common/config"
	"srp-go/common/msg"
	"strings"
)

func operate() {
	if serverTCP == nil {
		log.Error("no connect server")
		return
	}

	buf := make([]byte, 64*1024)

	fmt.Println("please input need operate")
	readLen, _ := os.Stdin.Read(buf)
	operateName := strings.TrimSpace(string(buf[:readLen]))

	fmt.Println("please input param")
	readLen, _ = os.Stdin.Read(buf)
	params := strings.TrimSpace(string(buf[:readLen]))

	_ = serverTCP.WriteSecurityMessage(&msg.OperateRequest{
		Operate: operateName,
		Params:  params,
	})
}

func doOperate(operateType string) {
	switch operateType {
	case "e":
		os.Exit(0)
	case "n":
		notify()
	case "o":
		operate()
	case "r":
		if err := config.RefreshConfig(); err != nil {
			log.Error("refresh config error", err)
		} else {
			closeServer()
		}
	case "c":
		connectInfo := ""
		for _, h := range clientHandles {
			connectInfo += h.ConnectInfo()
		}
		log.Info("\n",
			"connectInfo[\n",
			connectInfo,
			"]\n",
		)
	default:
		log.Warn("no command", operateType)
	}
}
