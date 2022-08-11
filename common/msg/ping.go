package msg

import "time"

type PingMessage struct {
	Date time.Time `json:"date"`
}

func (t *PingMessage) GetMessageType() uint32 {
	return PingMessageType
}

type PoneMessage struct {
	Date time.Time `json:"date"`
}

func (t *PoneMessage) GetMessageType() uint32 {
	return PoneMessageType
}
