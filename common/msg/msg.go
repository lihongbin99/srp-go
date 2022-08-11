package msg

import "fmt"

type Message interface {
	GetMessageType() uint32
}

const (
	_ uint32 = iota
	NewUDPType
	UDPPackageType
	UDPPackageConfirmType

	OperateRequestType
	OperateResponseType
	NotifyRequestType
	NotifyResponseType

	SrpRequestType

	SecurityRequestType
	SecurityResponseType
	InfoRequestType
	InfoResponseType
	PingMessageType
	PoneMessageType

	NatRegisterRequestType
	NatRegisterResponseType
	NatNewConnectRequestType
	NatNewConnectResponseType
	NatAnswerConnectRequestType
	NatAnswerConnectResponseType

	ProxyNewConnectRequestType
	ProxyNewConnectResponseType

	P2pNewConnectRequestType
	P2pNewConnectResponseType
)

func NewMessage(messageType uint32) (message Message, err error) {
	switch messageType {
	case NewUDPType:
		message = &NewUDP{}
	case UDPPackageType:
		message = &UDPPackage{}
	case UDPPackageConfirmType:
		message = &UDPPackageConfirm{}

	case OperateRequestType:
		message = &OperateRequest{}
	case OperateResponseType:
		message = &OperateResponse{}
	case NotifyRequestType:
		message = &NotifyRequest{}
	case NotifyResponseType:
		message = &NotifyResponse{}

	case SrpRequestType:
		message = &SrpRequest{}
	case SecurityRequestType:
		message = &SecurityRequest{}
	case SecurityResponseType:
		message = &SecurityResponse{}

	case InfoRequestType:
		message = &InfoRequest{}
	case InfoResponseType:
		message = &InfoResponse{}

	case PingMessageType:
		message = &PingMessage{}
	case PoneMessageType:
		message = &PoneMessage{}

	case NatRegisterRequestType:
		message = &NatRegisterRequest{}
	case NatRegisterResponseType:
		message = &NatRegisterResponse{}
	case NatNewConnectRequestType:
		message = &NatNewConnectRequest{}
	case NatNewConnectResponseType:
		message = &NatNewConnectResponse{}
	case NatAnswerConnectRequestType:
		message = &NatAnswerConnectRequest{}
	case NatAnswerConnectResponseType:
		message = &NatAnswerConnectResponse{}

	case ProxyNewConnectRequestType:
		message = &ProxyNewConnectRequest{}
	case ProxyNewConnectResponseType:
		message = &ProxyNewConnectResponse{}

	case P2pNewConnectRequestType:
		message = &P2pNewConnectRequest{}
	case P2pNewConnectResponseType:
		message = &P2pNewConnectResponse{}
	default:
		err = fmt.Errorf("no find message type: %d", messageType)
	}
	return
}
