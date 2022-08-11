package utils

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

type ID = uint64

var (
	idLock  = sync.Mutex{}
	idCache = make(map[ID]time.Time)
)

func RemoteId(id ID) {
	idLock.Lock()
	defer idLock.Unlock()
	delete(idCache, id)
}

func GetId() ID {
	idLock.Lock()
	defer idLock.Unlock()
	for {
		r := rand.Int63n(math.MaxInt64)
		if _, ok := idCache[ID(r)]; !ok {
			idCache[ID(r)] = time.Now()
			return ID(r)
		}
	}
}

func I2b32(i uint32) (buf []byte) {
	buf = make([]byte, 4, 4)
	buf[0] = byte(i >> 24)
	buf[1] = byte(i >> 16)
	buf[2] = byte(i >> 8)
	buf[3] = byte(i)
	return
}

func B2i32(buf []byte) (i uint32, err error) {
	if len(buf) >= 4 {
		i = uint32(buf[0])<<24 +
			uint32(buf[1])<<16 +
			uint32(buf[2])<<8 +
			uint32(buf[3])
	} else {
		err = fmt.Errorf("id byte arr len = %d", len(buf))
	}
	return
}

func I2b64(i ID) (buf []byte) {
	buf = make([]byte, 8, 8)
	buf[0] = byte(i >> 56)
	buf[1] = byte(i >> 48)
	buf[2] = byte(i >> 40)
	buf[3] = byte(i >> 32)
	buf[4] = byte(i >> 24)
	buf[5] = byte(i >> 16)
	buf[6] = byte(i >> 8)
	buf[7] = byte(i)
	return
}

func B2i64(buf []byte) (i ID, err error) {
	if len(buf) >= 8 {
		i = ID(buf[0])<<56 +
			ID(buf[1])<<48 +
			ID(buf[2])<<40 +
			ID(buf[3])<<32 +
			ID(buf[4])<<24 +
			ID(buf[5])<<16 +
			ID(buf[6])<<8 +
			ID(buf[7])
	} else {
		err = fmt.Errorf("id byte arr len = %d", len(buf))
	}
	return
}
