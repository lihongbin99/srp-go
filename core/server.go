package core

import (
	"srp-go/common/io"
	"sync"
)

var (
	TcpMap     = make(map[string]*io.TCP)
	TcpMapLock = sync.Mutex{}
)
