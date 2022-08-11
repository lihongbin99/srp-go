package p2p

import (
	"srp-go/common/config"
	"strings"
)

func (t *clientHandle) startService() {
	for serviceName, proxyConfig := range config.NewConfig.Proxy {
		// 解析协议
		if strings.Contains(proxyConfig.Protocol, "tcp") {
			clientConfig := &clientConfigTCP{proxyConfig, nil, "new"}
			t.configsTCP[serviceName] = clientConfig
			t.startTCP(serviceName, clientConfig)
		}
	}
}
