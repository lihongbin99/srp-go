package io

import (
	"fmt"
	"srp-go/common/logger"
	"srp-go/common/msg"
	"srp-go/common/utils"
	"sync"
	"time"
)

var (
	log = logger.NewLog("IO")

	timeout = 3 * time.Minute
)

type Message struct {
	Message msg.Message
	Err     error
}

type Base struct {
	ConnectInfo
	// 加密
	AesKey []byte
	AesIv  []byte

	// 读写数据
	Buf       []byte
	readLock  sync.Mutex
	writeLock sync.Mutex

	// 连接超时
	LastTransferTime time.Time
	TimeOut          bool
}

func newBase() *Base {
	return &Base{
		ConnectInfo{},
		nil,
		nil,
		make([]byte, 64*1024),
		sync.Mutex{},
		sync.Mutex{},
		time.Time{},
		false,
	}
}

type ConnectInfo struct {
	ServerId   utils.ID // 服务端的全局Id
	ClientName string
	Status     string
}

func (t *ConnectInfo) String() string {
	return fmt.Sprintf("{serverId: %d, ClientName: %s, status: %s}",
		t.ServerId, t.ClientName, t.Status)
}
