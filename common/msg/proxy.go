package msg

import "srp-go/common/utils"

type ProxyNewConnectRequest struct {
	Protocol   string   `json:"Protocol"`
	ClientName string   `json:"ClientName"`
	ClientAddr string   `json:"ClientAddr"`
	TargetName string   `json:"TargetName"`
	TargetIp   string   `json:"TargetIp"`
	TargetPort int      `json:"TargetPort"`
	ServerId   utils.ID `json:"ServerId"`
}

func (t *ProxyNewConnectRequest) GetMessageType() uint32 {
	return ProxyNewConnectRequestType
}

type ProxyNewConnectResponse struct {
	ServerId utils.ID `json:"ServerId"`
	Protocol string   `json:"Protocol"`
	Result   string   `json:"Result"`
}

func (t *ProxyNewConnectResponse) GetMessageType() uint32 {
	return ProxyNewConnectResponseType
}
