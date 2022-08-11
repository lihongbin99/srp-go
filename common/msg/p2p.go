package msg

import "srp-go/common/utils"

type P2pNewConnectRequest struct {
	Protocol   string   `json:"Protocol"`
	ClientName string   `json:"ClientName"`
	ClientAddr string   `json:"ClientAddr"`
	TargetName string   `json:"TargetName"`
	TargetIp   string   `json:"TargetIp"`
	TargetPort int      `json:"TargetPort"`
	ServerId   utils.ID `json:"ServerId"`
}

func (t *P2pNewConnectRequest) GetMessageType() uint32 {
	return P2pNewConnectRequestType
}

type P2pNewConnectResponse struct {
	ServerId utils.ID `json:"ServerId"`
	Protocol string   `json:"Protocol"`
	Result   string   `json:"Result"`
}

func (t *P2pNewConnectResponse) GetMessageType() uint32 {
	return P2pNewConnectResponseType
}
