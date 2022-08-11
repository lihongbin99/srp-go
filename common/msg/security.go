package msg

type SecurityRequest struct {
	KeyIv []byte `json:"keyIv"`
}

func (t *SecurityRequest) GetMessageType() uint32 {
	return SecurityRequestType
}

type SecurityResponse struct {
	Result string `json:"Result"`
}

func (t *SecurityResponse) GetMessageType() uint32 {
	return SecurityResponseType
}
