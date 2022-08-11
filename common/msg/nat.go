package msg

import "srp-go/common/utils"

type NatRegisterRequest struct {
	ServiceName string `json:"ServiceName"`
	Protocol    string `json:"Protocol"`
	RemotePort  int    `json:"RemotePort"`
	LocalPort   int    `json:"LocalPort"`
}

func (t *NatRegisterRequest) GetMessageType() uint32 {
	return NatRegisterRequestType
}

type NatRegisterResponse struct {
	ServiceName string `json:"ServiceName"`
	Protocol    string `json:"Protocol"`
	Result      string `json:"Result"`
}

func (t *NatRegisterResponse) GetMessageType() uint32 {
	return NatRegisterResponseType
}

type NatNewConnectRequest struct {
	ServiceName string   `json:"ServiceName"`
	Protocol    string   `json:"Protocol"`
	ConnectId   utils.ID `json:"ConnectId"`
	ClientAddr  string   `json:"ClientAddr"`
}

func (t *NatNewConnectRequest) GetMessageType() uint32 {
	return NatNewConnectRequestType
}

type NatNewConnectResponse struct {
	ServiceName string   `json:"ServiceName"`
	Protocol    string   `json:"Protocol"`
	ConnectId   utils.ID `json:"ConnectId"`
	Result      string   `json:"Result"`
}

func (t *NatNewConnectResponse) GetMessageType() uint32 {
	return NatNewConnectResponseType
}

type NatAnswerConnectRequest struct {
	ServiceName string   `json:"ServiceName"`
	Protocol    string   `json:"Protocol"`
	ConnectId   utils.ID `json:"ConnectId"`
}

func (t *NatAnswerConnectRequest) GetMessageType() uint32 {
	return NatAnswerConnectRequestType
}

type NatAnswerConnectResponse struct {
	ServiceName string   `json:"ServiceName"`
	Protocol    string   `json:"Protocol"`
	ConnectId   utils.ID `json:"ConnectId"`
	Result      string   `json:"Result"`
}

func (t *NatAnswerConnectResponse) GetMessageType() uint32 {
	return NatAnswerConnectResponseType
}
