package p2p

import (
	"fmt"
	"net"
	"srp-go/common/config"
	"srp-go/common/io"
)

type clientConfigTCP struct {
	*config.ProxyConfig
	listen *net.TCPListener
	status string
}

type clientConnectTCP struct {
	thisName   string
	targetName string
	localAddr  string
	targetAddr string

	remoteTCP *io.TCP
	localTCP  *net.TCPConn
	status    string
}

func (t *clientConnectTCP) String() string {
	return fmt.Sprintf("tcp {thisName: %s, targetName: %s, localAddr: %s, targetAddr: %s, status: %s}", t.thisName, t.targetName, t.localAddr, t.targetAddr, t.status)
}
