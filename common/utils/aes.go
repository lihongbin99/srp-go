package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"math/rand"
)

var (
	aesRandomUUID = []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")
)

func getAesRandomUUID(n int) []byte {
	result := make([]byte, n)
	for i := 0; i < n; i++ {
		result[i] = aesRandomUUID[rand.Intn(len(aesRandomUUID))]
	}
	return result
}

func GenerateAES() (key, iv []byte) {
	key = getAesRandomUUID(16)
	iv = getAesRandomUUID(16)
	return
}

func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padText...)
}

func PKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unPadding := int(origData[length-1])
	return origData[:(length - unPadding)]
}

func AesEncrypt(plaintext []byte, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	plaintext = PKCS7Padding(plaintext, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, iv)
	crypt := make([]byte, len(plaintext))
	blockMode.CryptBlocks(crypt, plaintext)
	return crypt, nil
}

func AesDecrypt(ciphertext []byte, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, iv[:blockSize])
	origData := make([]byte, len(ciphertext))
	blockMode.CryptBlocks(origData, ciphertext)
	origData = PKCS7UnPadding(origData)
	return origData, nil
}
