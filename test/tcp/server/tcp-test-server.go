package main

import (
	"net"
	"srp-go/common/logger"
	"time"
)

var (
	log = logger.NewLog("Server-Test")
)

func main() {
	addr, err := net.ResolveTCPAddr("tcp", ":28080")
	if err != nil {
		log.Error(err)
		return
	}

	listen, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Error(err)
		return
	}

	for {
		tcp, err := listen.AcceptTCP()
		if err != nil {
			log.Error(err)
			return
		}

		go test(tcp)
	}
}

func test(tcp *net.TCPConn) {
	buf := make([]byte, 13)
	for i := 0; i < 3; i++ {
		readLength, err := tcp.Read(buf)
		if err != nil {
			log.Error(err)
			return
		}
		log.Info("Test", i, ":", string(buf[:readLength]))
	}

	buf = make([]byte, 64*1024)

	// 下载测速
	maxRead := 0
	readLength, err := tcp.Read(buf)
	if err != nil {
		log.Error(err)
		return
	}
	log.Info("Start")
	maxRead += readLength
	startTime := time.Now()
	endTime := time.Now()

	for {
		_ = tcp.SetReadDeadline(time.Now().Add(3 * time.Second))
		readLength, err = tcp.Read(buf)
		if err != nil {
			log.Error(err)
			break
		}
		_ = tcp.SetReadDeadline(time.Time{})
		maxRead += readLength
		endTime = time.Now()
	}

	v := endTime.Sub(startTime) / time.Second
	log.Info("maxDownload:", maxRead)
	log.Info("v:", int(v))
	if v > 0 {
		log.Info("Download:", maxRead/1024/1024/int(v), "MB/s")
	}

	// 上传测速
	time.Sleep(1 * time.Second)

	maxWrite := 0
	c := true
	ticker := time.NewTicker(3 * time.Second)
	for c {
		select {
		case _ = <-ticker.C:
			c = false
			break
		default:
			writeLength, err := tcp.Write(buf)
			if err != nil {
				log.Error(err)
				return
			}
			maxWrite += writeLength
			//time.Sleep(10 * time.Millisecond)
		}
	}

	log.Info("maxUpload:", maxWrite)
	log.Info("Upload:", maxWrite/1024/1024/3, "MB/s")
}
