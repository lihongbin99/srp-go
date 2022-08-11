package msg

import (
	"srp-go/common/utils"
	"time"
)

type NewUDP struct {
	Id utils.ID `json:"id"`
}

func (t *NewUDP) GetMessageType() uint32 {
	return NewUDPType
}

type UDPPackage struct {
	M        utils.ID // 消息id
	L        uint32   // 消息长度
	O        uint32   // 起始位置偏移量
	S        uint32   // 已读长度
	T        ClientType
	D        []byte // 消息
	LastTime time.Time
	OMap     map[uint32]uint32
	Stop     bool
}

func (t *UDPPackage) GetMessageType() uint32 {
	return UDPPackageType
}

type UDPPackageConfirm struct {
	M utils.ID // 消息id
	O uint32   // 起始位置偏移量
}

func (t *UDPPackageConfirm) GetMessageType() uint32 {
	return UDPPackageConfirmType
}
