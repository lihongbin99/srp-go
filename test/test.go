package main

import (
	"os"
	"srp-go/common/logger"
	"srp-go/common/utils"
)

var (
	log = logger.NewLog("Test")
)

func main() {
	data, err := os.ReadFile("C:\\Users\\Lee\\Videos\\test\\4ffae39b723d8030ff3636ea38efecb5_2_0.ts")
	if err != nil {
		log.Error(err)
		return
	}

	key := []byte{240, 180, 53, 1, 85, 194, 178, 17, 227, 47, 125, 205, 49, 77, 251, 61}
	iv := []byte{0x96, 0x83, 0xed, 0x41, 0xef, 0x6a, 0x89, 0xec, 0xa5, 0x3b, 0xc2, 0xb9, 0x66, 0x51, 0xac, 0x15}

	video, err := utils.AesDecrypt(data, key, iv)
	if err != nil {
		log.Error(err)
		return
	}

	err = os.WriteFile("C:\\Users\\Lee\\Videos\\test\\01.ts", video, 0x666)
	if err != nil {
		log.Error(err)
	}

}
