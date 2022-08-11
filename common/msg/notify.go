package msg

type NotifyRequest struct {
	OperateName string `json:"OperateName"`
	Operate     string `json:"Operate"`
	ClientName  string `json:"ClientName"`
	Params      string `json:"Params"`
}

func (t *NotifyRequest) GetMessageType() uint32 {
	return NotifyRequestType
}

type NotifyResponse struct {
	OperateName string `json:"OperateName"`
	Result      string `json:"Result"`
}

func (t *NotifyResponse) GetMessageType() uint32 {
	return NotifyResponseType
}
