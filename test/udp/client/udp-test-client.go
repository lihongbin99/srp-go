package main

import (
	"net"
	"srp-go/common/logger"
	"strconv"
	"time"
)

var (
	log = logger.NewLog("Client-Test")
)

func main() {
	count := 3
	over := make(chan interface{})
	for i := 0; i < count; i++ {
		go test(over)
	}
	for i := 0; i < count; i++ {
		<-over
	}
}

func test(over chan interface{}) {
	//addr, err := net.ResolveUDPAddr("udp", "0.0.0.0:23521")// Nat
	addr, err := net.ResolveUDPAddr("udp", "0.0.0.0:23523") // Proxy
	//addr, err := net.ResolveUDPAddr("udp", "0.0.0.0:28080")// Src
	if err != nil {
		log.Error(err)
		return
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Error(err)
		return
	}
	for i := 1; i <= 3; i++ {
		_, err = conn.Write([]byte("Hello World-" + strconv.Itoa(i)))
		if err != nil {
			log.Error(err)
			return
		}
	}

	// 上传测速
	time.Sleep(1 * time.Second)

	buf := make([]byte, 32*1024)

	maxWrite := 0
	c := true
	ticker := time.NewTicker(3 * time.Second)
	for c {
		select {
		case _ = <-ticker.C:
			c = false
			break
		default:
			writeLength, err := conn.Write(buf)
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

	// 下载测速
	maxRead := 0
	readLength, err := conn.Read(buf)
	if err != nil {
		log.Error(err)
		return
	}
	log.Info("Start")
	maxRead += readLength
	startTime := time.Now()
	endTime := time.Now()

	for {
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		readLength, err = conn.Read(buf)
		if err != nil {
			log.Error(err)
			break
		}
		_ = conn.SetReadDeadline(time.Time{})
		maxRead += readLength
		endTime = time.Now()
	}

	v := endTime.Sub(startTime) / time.Second
	log.Info("maxDownload:", maxRead)
	log.Info("v", int(v))
	log.Info("Download:", maxRead/1024/1024/int(v), "MB/s")

	over <- 1
}
