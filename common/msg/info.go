package msg

import "srp-go/common/utils"

type ClientType uint8

const (
	_ ClientType = iota
	ClientTypeRegister
	ClientTypeNat
	ClientTypeProxy
	ClientTypeP2p
)

type InfoRequest struct {
	ClientName  string     `json:"ClientName"`
	Version     string     `json:"Version"`
	ConnectType ClientType `json:"ConnectType"`
}

func (t *InfoRequest) GetMessageType() uint32 {
	return InfoRequestType
}

type InfoResponse struct {
	Version  string   `json:"Version"`
	ServerId utils.ID `json:"ServerId"`
	Result   string   `json:"Result"`
}

func (t *InfoResponse) GetMessageType() uint32 {
	return InfoResponseType
}
