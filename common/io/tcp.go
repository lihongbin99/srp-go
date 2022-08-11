package io

import (
	"fmt"
	"net"
	"srp-go/common/config"
	"srp-go/common/msg"
	"srp-go/common/utils"
	"time"
)

type TCP struct {
	*net.TCPConn
	*Base
}

func NewTCP(conn *net.TCPConn) *TCP {
	t := &TCP{conn, newBase()}
	return t
}

func (t *TCP) ReadSecurity() error {
	if config.SecurityPrivateKey == nil || len(config.SecurityPrivateKey) <= 0 {
		return fmt.Errorf("no private key")
	}
	t.readLock.Lock()
	defer t.readLock.Unlock()
	messageLen := 256 // RAS加密后的长度
	maxReadLength := 0
	for maxReadLength < messageLen {
		_ = t.TCPConn.SetReadDeadline(time.Now().Add(3 * time.Second))
		readLength, err := t.TCPConn.Read(t.Buf[maxReadLength : messageLen-maxReadLength])
		if err != nil {
			return err
		}
		_ = t.TCPConn.SetReadDeadline(time.Time{})
		maxReadLength += readLength
	}

	message, err := utils.DecryptRSA(t.Buf[:256], config.SecurityPrivateKey)
	if err != nil {
		return err
	}
	if len(message) != 32 { // AES的key+iv的长度
		return fmt.Errorf("len(key+iv) = %d", len(message))
	}
	t.AesKey = message[:16]
	t.AesIv = message[16:]

	return nil
}

func (t *TCP) WriteSecurity() error {
	if config.SecurityPublicKey == nil || len(config.SecurityPublicKey) <= 0 {
		return fmt.Errorf("no public key")
	}

	key, iv := utils.GenerateAES()
	t.AesKey = key
	t.AesIv = iv

	t.writeLock.Lock()
	defer t.writeLock.Unlock()
	message := make([]byte, len(key)+len(iv))
	copy(message[0:len(key)], key)
	copy(message[len(key):], iv)
	message, err := utils.EncryptRSA(message, config.SecurityPublicKey)
	if err != nil {
		return err
	}
	_, err = t.TCPConn.Write(message)
	if err != nil {
		return err
	}
	return nil
}

func (t *TCP) WriteMessage(message msg.Message) error {
	return t.write(message, false)
}

func (t *TCP) WriteSecurityMessage(message msg.Message) error {
	return t.write(message, true)
}

func (t *TCP) write(message msg.Message, security bool) error {
	t.writeLock.Lock()
	defer t.writeLock.Unlock()

	// 解析
	data, err := msg.ToByte(message)
	if err != nil {
		return err
	}
	if len(data) <= 0 {
		return nil
	}

	// 加密
	if security && config.NewConfig.Common.Security.Enable {
		if data, err = utils.AesEncrypt(data, t.AesKey, t.AesIv); err != nil {
			return err
		}
	}

	// 发送消息类型
	if _, err = t.TCPConn.Write(utils.I2b32(message.GetMessageType())); err != nil {
		return err
	}
	// 发送消息长度
	if _, err = t.TCPConn.Write(utils.I2b32(uint32(len(data)))); err != nil {
		return err
	}
	// 发送消息
	if _, err = t.TCPConn.Write(data); err != nil {
		return err
	}
	return nil
}

func (t *TCP) ReadMessage(timeout time.Time) Message {
	return t.read(timeout, false)
}

func (t *TCP) ReadSecurityMessage(timeout time.Time) Message {
	return t.read(timeout, true)
}

func (t *TCP) read(timeout time.Time, security bool) Message {
	t.readLock.Lock()
	defer t.readLock.Unlock()

	_ = t.SetReadDeadline(timeout)

	// 读取前缀
	readSum := 0
	for readSum < 8 {
		if readLength, err := t.TCPConn.Read(t.Buf[readSum:8]); err != nil {
			return Message{Err: err}
		} else {
			readSum += readLength
		}
	}

	// 获取消息类型
	messageType, err := utils.B2i32(t.Buf[:4])
	if err != nil {
		return Message{Err: err}
	}
	message, err := msg.NewMessage(messageType)
	if err != nil {
		return Message{Err: err}
	}

	// 获取消息长度
	messageLen32, err := utils.B2i32(t.Buf[4:8])
	if err != nil {
		return Message{Err: err}
	}
	messageLen := int(messageLen32)
	if messageLen <= 0 {
		return Message{Err: err}
	}
	if messageLen > len(t.Buf) {
		return Message{Err: fmt.Errorf("message len: %d", messageLen)}
	}

	// 读取消息
	readSum = 0
	for readSum < messageLen {
		if readLength, err := t.TCPConn.Read(t.Buf[readSum : messageLen-readSum]); err != nil {
			return Message{Err: err}
		} else {
			readSum += readLength
		}
	}
	data := t.Buf[:messageLen]

	// 解密
	if security && config.NewConfig.Common.Security.Enable {
		if data, err = utils.AesDecrypt(data, t.AesKey, t.AesIv); err != nil {
			return Message{Err: err}
		}
	}

	// 解析
	if err = msg.ToObj(data, message); err != nil {
		return Message{Err: err}
	}

	_ = t.SetReadDeadline(time.Time{})
	return Message{Message: message, Err: nil}
}

func (t *TCP) ServerInit() (err error, connectType msg.ClientType) {
	t.ServerId = utils.GetId()
	t.Status = "start init"

	// 1. 设置协议
	_ = t.WriteMessage(&msg.SrpRequest{
		Project:  "srp",
		Version:  config.Version,
		Security: config.NewConfig.Common.Security.Enable,
	})

	// 2. 设置加密
	if config.NewConfig.Common.Security.Enable {
		if err = t.ReadSecurity(); err != nil {
			_ = t.WriteMessage(&msg.SecurityResponse{Result: err.Error()})
			err = fmt.Errorf("read security error: %s", err.Error())
			return
		}
		_ = t.WriteMessage(&msg.SecurityResponse{Result: "success"})
	}

	// 3. 获取基础信息
	message := t.ReadSecurityMessage(time.Now().Add(8 * time.Second))
	if message.Err != nil {
		err = fmt.Errorf("read info error: %s", message.Err)
		return
	}
	infoRequest, convertResult := message.Message.(*msg.InfoRequest)
	if !convertResult {
		err = fmt.Errorf("read info type error: %d", message.Message.GetMessageType())
		return
	}
	if err = config.VerifyVersion(infoRequest.Version); err != nil {
		_ = t.WriteSecurityMessage(&msg.InfoResponse{
			Version:  config.Version,
			ServerId: t.ServerId,
			Result:   err.Error(),
		})
		return
	}
	t.ClientName = infoRequest.ClientName

	t.Status = "init success"
	return nil, infoRequest.ConnectType
}

func (t *TCP) ClientInit(connectType msg.ClientType) error {
	t.Status = "start init"

	// 1. 获取协议
	message := t.ReadMessage(time.Now().Add(8 * time.Second))
	if message.Err != nil {
		return fmt.Errorf("read srp error: %s", message.Err)
	}
	srpRequest, convertResult := message.Message.(*msg.SrpRequest)
	if !convertResult {
		return fmt.Errorf("read srp type error: %d", message.Message.GetMessageType())
	}
	// 校验版本
	if err := config.VerifyVersion(srpRequest.Version); err != nil {
		return fmt.Errorf("verify version error: %s", err)
	}
	if srpRequest.Security != config.NewConfig.Common.Security.Enable {
		return fmt.Errorf("security error, client: %v, server: %v", config.NewConfig.Common.Security.Enable, srpRequest.Security)
	}

	// 2. 设置加密
	if config.NewConfig.Common.Security.Enable {
		_ = t.WriteSecurity()
		message = t.ReadMessage(time.Now().Add(8 * time.Second))
		if message.Err != nil {
			return fmt.Errorf("read security error: %s", message.Err)
		}
		securityResponse, convertResult := message.Message.(*msg.SecurityResponse)
		if !convertResult {
			return fmt.Errorf("read security type error: %d", message.Message.GetMessageType())
		}
		if securityResponse.Result != "success" {
			return fmt.Errorf("security error: %s", securityResponse.Result)
		}
	}

	// 3. 设置基础信息
	t.ClientName = config.NewConfig.Common.Client.Name
	_ = t.WriteSecurityMessage(&msg.InfoRequest{
		ClientName:  t.ClientName,
		Version:     config.Version,
		ConnectType: connectType,
	})

	message = t.ReadSecurityMessage(time.Now().Add(8 * time.Second))
	if message.Err != nil {
		return fmt.Errorf("read info error: %s", message.Err)
	}
	infoResponse, convertResult := message.Message.(*msg.InfoResponse)
	if !convertResult {
		return fmt.Errorf("read info type error: %d", message.Message.GetMessageType())
	}
	t.ServerId = infoResponse.ServerId
	if infoResponse.Result != "success" {
		return fmt.Errorf("info verify error: %s", infoResponse.Result)
	}

	t.Status = "init success"
	return nil
}

func (t *TCP) Transfer(dest, src *net.TCPConn, finish chan interface{}) {
	defer func() {
		_ = src.Close()
		_ = dest.Close()
	}()
	buf := make([]byte, 64*1024)
	for {
		length, err := src.Read(buf)
		if t.TimeOut {
			break
		}
		if err != nil {
			break
		}
		if _, err = dest.Write(buf[:length]); err != nil {
			break
		}
		t.LastTransferTime = time.Now()
	}

	finish <- 1
}

func (t *TCP) GoTimeOut() {
	time.Sleep(timeout)
	for {
		if time.Now().Sub(t.LastTransferTime) > timeout {
			break
		}
		time.Sleep(t.LastTransferTime.Add(timeout).Sub(time.Now()))
	}
	t.TimeOut = true
	_ = t.Close()
}
