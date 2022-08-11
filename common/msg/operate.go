package msg

type OperateRequest struct {
	Operate string `json:"Operate"`
	Params  string `json:"Params"`
}

func (t *OperateRequest) GetMessageType() uint32 {
	return OperateRequestType
}

type OperateResponse struct {
	Result string `json:"Result"`
}

func (t *OperateResponse) GetMessageType() uint32 {
	return OperateResponseType
}
