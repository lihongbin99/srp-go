package main

import (
	"net"
	"srp-go/common/logger"
	"time"
)

var (
	log = logger.NewLog("Server-Test")

	hello = make(map[string]int)
	st    = make(map[string]time.Time)
	et    = make(map[string]time.Time)
)

func main() {
	addr, err := net.ResolveUDPAddr("udp", ":28080")
	if err != nil {
		log.Error(err)
		return
	}

	listen, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Error(err)
		return
	}

	buf := make([]byte, 64*1024)
	for {
		readLen, remoteAddr, err := listen.ReadFromUDP(buf)
		if err != nil {
			log.Error(err)
			return
		}
		if c, ok := hello[remoteAddr.String()]; !ok {
			log.Info("Start")
			st[remoteAddr.String()] = time.Now()
			c = 1
			hello[remoteAddr.String()] = c
			log.Info("Test", c, ":", string(buf[:readLen]))
		} else if c < 3 {
			c++
			hello[remoteAddr.String()] = c
			log.Info("Test", c, ":", string(buf[:readLen]))
		} else if c == 3 {
			c += readLen
			hello[remoteAddr.String()] = c
			go test(listen, remoteAddr)
		} else {
			c += readLen
			hello[remoteAddr.String()] = c
			et[remoteAddr.String()] = time.Now()
		}
	}
}

func test(listen *net.UDPConn, remoteAddr *net.UDPAddr) {
	for {
		c1 := hello[remoteAddr.String()]
		time.Sleep(8 * time.Second)
		c2 := hello[remoteAddr.String()]
		if c1 == c2 {
			break
		}
	}

	startTime := st[remoteAddr.String()]
	endTime := et[remoteAddr.String()]

	v := endTime.Sub(startTime) / time.Second
	log.Info("maxDownload:", hello[remoteAddr.String()]-3)
	log.Info("v:", int(v))
	if v > 0 {
		log.Info("Download:", (hello[remoteAddr.String()]-3)/1024/1024/int(v), "MB/s")
	}

	// 上传测速
	time.Sleep(1 * time.Second)

	maxWrite := 0
	c := true
	ticker := time.NewTicker(3 * time.Second)
	buf := make([]byte, 32*1024)
	for c {
		select {
		case _ = <-ticker.C:
			c = false
			break
		default:
			writeLength, err := listen.WriteToUDP(buf, remoteAddr)
			if err != nil {
				log.Error(err)
				return
			}
			maxWrite += writeLength
			time.Sleep(10 * time.Millisecond)
		}
	}

	log.Info("maxUpload:", maxWrite)
	log.Info("Upload:", maxWrite/1024/1024/3, "MB/s")
}
