package main

import (
	"fmt"
	"os"
	"srp-go/common/io"
	"srp-go/common/msg"
	"strings"
)

func notify() {
	if serverTCP == nil {
		log.Error("no connect server")
		return
	}

	buf := make([]byte, 64*1024)

	fmt.Println("please input need operate")
	readLen, _ := os.Stdin.Read(buf)
	operate := strings.TrimSpace(string(buf[:readLen]))

	fmt.Println("please input need operate client name")
	readLen, _ = os.Stdin.Read(buf)
	clientName := strings.TrimSpace(string(buf[:readLen]))

	fmt.Println("please input param")
	readLen, _ = os.Stdin.Read(buf)
	params := strings.TrimSpace(string(buf[:readLen]))

	_ = serverTCP.WriteSecurityMessage(&msg.NotifyRequest{
		Operate:    operate,
		ClientName: clientName,
		Params:     params,
	})
}

func doNotify(tcp *io.TCP, message *msg.NotifyRequest) {
	log.Info("receiver notify request", message.OperateName, message.Operate, message.Params)
	switch message.Operate {
	case "e":
		_ = tcp.WriteSecurityMessage(&msg.NotifyResponse{
			OperateName: message.OperateName,
			Result:      "close success",
		})
		doOperate(message.Operate)
	case "r":
		doOperate(message.Operate)
	default:
		_ = tcp.WriteSecurityMessage(&msg.NotifyResponse{
			OperateName: message.OperateName,
			Result:      fmt.Sprintf("operate error: %s", message.Operate),
		})
	}
}
