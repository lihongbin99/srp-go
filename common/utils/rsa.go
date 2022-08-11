package utils

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
)

func GenerateRSA() (privateKey, publicKey []byte, err error) {
	RSA, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return
	}

	privateByte := x509.MarshalPKCS1PrivateKey(RSA)
	publicByte, err := x509.MarshalPKIXPublicKey(&RSA.PublicKey)
	if err != nil {
		return
	}

	privateBlock := &pem.Block{
		Type:  "private key",
		Bytes: privateByte,
	}
	publicBlock := &pem.Block{
		Type:  "public key",
		Bytes: publicByte,
	}

	privateBuffer := &bytes.Buffer{}
	publicBuffer := &bytes.Buffer{}
	if err = pem.Encode(privateBuffer, privateBlock); err != nil {
		return
	}
	if err = pem.Encode(publicBuffer, publicBlock); err != nil {
		return
	}

	return privateBuffer.Bytes(), publicBuffer.Bytes(), nil
}

func GenerateRSAFile(privateFilePath, publicFilePath string) error {
	// filename.pem
	privateKey, publicKey, err := GenerateRSA()
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(privateFilePath, privateKey, 0666); err != nil {
		return err
	}
	if err = ioutil.WriteFile(publicFilePath, publicKey, 0666); err != nil {
		return err
	}
	return nil
}

func EncryptRSA(message, publicKey []byte) (result []byte, err error) {
	publicBlock, _ := pem.Decode(publicKey)

	keyInit, err := x509.ParsePKIXPublicKey(publicBlock.Bytes)
	if err != nil {
		return
	}

	RSA := keyInit.(*rsa.PublicKey)
	result, err = rsa.EncryptPKCS1v15(rand.Reader, RSA, message)
	return
}

func DecryptRSA(message, privateKey []byte) (result []byte, err error) {
	block, _ := pem.Decode(privateKey)

	RSA, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return
	}

	result, err = rsa.DecryptPKCS1v15(rand.Reader, RSA, message)
	return
}
