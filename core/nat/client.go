package nat

import (
	"fmt"
	"srp-go/common/config"
	"srp-go/common/io"
	"srp-go/common/msg"
	"strings"
)

func (t *clientHandle) registerService() {
	for serviceName, natConfig := range config.NewConfig.Nat {
		if _, ok := t.configs[serviceName]; !ok {
			t.configs[serviceName] = make(map[string]clientConfig)
		}
		// 解析协议
		if strings.Contains(natConfig.Protocol, "tcp") {
			t.configs[serviceName]["tcp"] = clientConfig{natConfig, "new"}
			t.sendRegisterService(serviceName, natConfig, "tcp")
		}
		if strings.Contains(natConfig.Protocol, "udp") {
			t.configs[serviceName]["udp"] = clientConfig{natConfig, "new"}
			t.sendRegisterService(serviceName, natConfig, "udp")
		}
	}
}

func (t *clientHandle) sendRegisterService(serviceName string, natConfig *config.NatConfig, protocol string) {
	if t.serverTCP != nil {
		_ = t.serverTCP.WriteSecurityMessage(&msg.NatRegisterRequest{
			ServiceName: serviceName,
			Protocol:    protocol,
			RemotePort:  natConfig.RemotePort,
			LocalPort:   natConfig.LocalPort,
		})
	}
}

func (t *clientHandle) registerResponse(_ *io.TCP, message *msg.NatRegisterResponse) {
	if message.Result != "success" {
		log.Warn("register", message.ServiceName, message.Protocol, "error:", message.Result)
		return
	}
	if cp, ok := t.configs[message.ServiceName]; ok {
		if c, ok := cp[message.Protocol]; ok {
			c.status = "run"
			log.Info("register", message.ServiceName, message.Protocol, "success", fmt.Sprintf("%s:%d", config.NewConfig.Common.Server.Ip, c.RemotePort), "->", fmt.Sprintf(":%d", c.LocalPort))
			return
		}
	}
	log.Warn("register", message.ServiceName, message.Protocol, "success, but no find service_name or protocol by configs")
}

func (t *clientHandle) newConnectRequest(tcp *io.TCP, message *msg.NatNewConnectRequest) {
	cp, ok := t.configs[message.ServiceName]
	if !ok {
		_ = tcp.WriteSecurityMessage(&msg.NatNewConnectResponse{
			ServiceName: message.ServiceName,
			Protocol:    message.Protocol,
			ConnectId:   message.ConnectId,
			Result:      fmt.Sprintf("not find service_name by %s", message.ServiceName),
		})
		return
	}
	c, ok := cp[message.Protocol]
	if !ok {
		_ = tcp.WriteSecurityMessage(&msg.NatNewConnectResponse{
			ServiceName: message.ServiceName,
			Protocol:    message.Protocol,
			ConnectId:   message.ConnectId,
			Result:      fmt.Sprintf("not find protocol by %s-%s", message.ServiceName, message.Protocol),
		})
		return
	}

	if message.Protocol == "tcp" {
		go t.newConnectTCP(tcp, c, message.ServiceName, message.ConnectId, message.ClientAddr)
	} else if message.Protocol == "udp" {
		go t.newConnectUDP(tcp, c, message.ServiceName, message.ConnectId, message.ClientAddr)
	} else {
		_ = tcp.WriteSecurityMessage(&msg.NatNewConnectResponse{
			ServiceName: message.ServiceName,
			Protocol:    message.Protocol,
			ConnectId:   message.ConnectId,
			Result:      fmt.Sprintf("protocol error: %s", c.Protocol),
		})
	}
}
