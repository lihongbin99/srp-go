package nat

import (
	"fmt"
	"net"
	"srp-go/common/io"
	"srp-go/common/utils"
)

type serverConfig interface {
	closeConfig()
	String() string
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type serverConfigTCP struct {
	*net.TCPListener
	serverId    utils.ID
	serviceName string
	protocol    string
	remotePort  int
	localPort   int
}

func (t *serverConfigTCP) closeConfig() {
	_ = t.Close()
}
func (t *serverConfigTCP) String() string {
	return fmt.Sprintf("{serverId: %d, serviceName: %s, protocol: %s, remotePort: %d, localPort: %d}", t.serverId, t.serviceName, t.protocol, t.remotePort, t.localPort)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type serverConfigConnectTCP struct {
	*serverConfigTCP
	clientTCP *io.TCP

	connectId     utils.ID
	clientConnect *net.TCPConn
	serverConnect *net.TCPConn

	targetAddr string
	status     string
}

func (t *serverConfigConnectTCP) Close() {
	_ = t.clientConnect.Close()
	if t.serverConnect != nil {
		_ = t.serverConnect.Close()
	}
	utils.RemoteId(t.connectId)
}
func (t *serverConfigConnectTCP) String() string {
	return fmt.Sprintf("tcp {serverId: %d, serviceName: %s, protocol: %s, remotePort: %d, connetId: %d, targetAddr: %s, status: %s}", t.serverId, t.serviceName, t.protocol, t.remotePort, t.connectId, t.targetAddr, t.status)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type serverConfigUDP struct {
	*net.UDPConn
	serverId    utils.ID
	serviceName string
	protocol    string
	remotePort  int
}

func (t *serverConfigUDP) closeConfig() {
	_ = t.Close()
}
func (t *serverConfigUDP) String() string {
	return fmt.Sprintf("{serverId: %d, serviceName: %s, protocol: %s, remotePort: %d}", t.serverId, t.serviceName, t.protocol, t.remotePort)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type serverConfigConnectUDP struct {
	*serverConfigUDP
	clientTCP *io.TCP

	connectId     utils.ID
	clientAddr    *net.UDPAddr
	clientConnect *io.UDP
	serverAddr    *net.UDPAddr
	serverConnect *io.UDP

	cache [][]byte // UDP 缓存

	targetAddr string
	status     string
}

func (t *serverConfigConnectUDP) ClientClose() {
	if t.clientConnect != nil {
		_ = t.clientConnect.Close()
	}
	if t.serverConnect != nil {
		_ = t.serverConnect.Close()
	}
}
func (t *serverConfigConnectUDP) String() string {
	return fmt.Sprintf("udp {serverId: %d, serviceName: %s, protocol: %s, remotePort: %d, connetId: %d, targetAddr: %s, status: %s}", t.serverId, t.serviceName, t.protocol, t.remotePort, t.connectId, t.targetAddr, t.status)
}
