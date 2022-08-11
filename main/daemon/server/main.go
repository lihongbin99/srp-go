package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	srpName    = "./frps"
	newSrpName = "./frps.new"
	startBat   = "./frps.sh"
	daemonName = "frps_d"
)

func main() {
	for {
		time.Sleep(1 * time.Minute)

		buf := bytes.Buffer{}
		cmd1 := exec.Command("/bin/sh", "-c", "ps -ef | grep "+srpName+" | grep -v "+daemonName+" | grep -v grep")
		cmd1.Stdout = &buf
		if err := cmd1.Run(); err != nil {
			restart()
			continue
		}

		if !strings.Contains(string(buf.Bytes()), srpName) {
			restart()
		}
	}
}

func restart() {
	fmt.Println("restart")

	// 替换文件
	file, err := ioutil.ReadFile(newSrpName)
	if err == nil {
		_ = os.Remove(srpName)
		if err := ioutil.WriteFile(srpName, file, 0700); err != nil {
			fmt.Println("write ", srpName, " error", err)
			return
		}
		fmt.Println("write", srpName, "success")
	}

	// 检测启动文件是否存在
	if _, err = os.Stat(startBat); err != nil {
		fmt.Println("not find bat", startBat)
		return
	}

	// 启动文件
	cmd := exec.Command("/bin/sh", "-c", startBat)
	if err = cmd.Run(); err != nil {
		fmt.Println("start error", err)
		return
	}

	fmt.Println("start success")
}
