package msg

type SrpRequest struct {
	Project  string `json:"Project"`
	Version  string `json:"Version"`
	Security bool   `json:"Security"`
}

func (t *SrpRequest) GetMessageType() uint32 {
	return SrpRequestType
}
