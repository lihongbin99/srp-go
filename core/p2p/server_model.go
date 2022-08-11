package p2p

import (
	"fmt"
	"srp-go/common/io"
	"srp-go/common/utils"
)

type serverConnectTCP struct {
	ccTCP    *io.TCP
	csTCP    *io.TCP
	waitChan chan bool

	connectId  utils.ID
	ccName     string
	ccAddr     string
	csName     string
	csAddr     string
	targetAddr string
	status     string
}

func (t *serverConnectTCP) String() string {
	return fmt.Sprintf("tcp {connectId: %d, ccName: %s, ccAddr: %s, csName: %s csAddr: %s, targetAddr: %s status: %s}", t.connectId, t.ccName, t.ccAddr, t.csName, t.csAddr, t.targetAddr, t.status)
}
