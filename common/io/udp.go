package io

import (
	"fmt"
	"net"
	"srp-go/common/config"
	"srp-go/common/msg"
	"srp-go/common/utils"
	"sync"
	"time"
)

var (
	//maxPackageLen     = 64 * 1024
	maxPackageLen = 1464
	//maxPackageLen     = 548
	maxPackageDataLen = maxPackageLen - 21

	udpPackageCache     = make(map[utils.ID]*msg.UDPPackage)
	udpPackageCacheLock = sync.Mutex{}

	writeUDPCacheMap  = make(map[utils.ID]map[uint32]*writeUDPCache)
	writeUDPCacheLock = sync.Mutex{}
)

func init() {
	go func() {
		ticker := time.NewTicker(8 * time.Second)
		for {
			select {
			case <-ticker.C:
				flushCache()
			}
		}
	}()
}

func flushCache() {
	udpPackageCacheLock.Lock()
	defer udpPackageCacheLock.Unlock()
	t := time.Now().Add(-8 * time.Second)
	for m, c := range udpPackageCache {
		if c.LastTime.Before(t) {
			delete(udpPackageCache, m)
		}
	}
}

type writeUDPCache struct {
	conn *net.UDPConn
	addr *net.UDPAddr
	buf  []byte
}

type UDP struct {
	*net.UDPConn
	*Base
	Security bool
}

func NewUDP(conn *net.UDPConn) *UDP {
	u := &UDP{conn, newBase(), false}
	return u
}

func (t *UDP) ReadSecurity(m *msg.SecurityRequest) error {
	if config.SecurityPrivateKey == nil || len(config.SecurityPrivateKey) <= 0 {
		return fmt.Errorf("no private key")
	}

	message, err := utils.DecryptRSA(m.KeyIv, config.SecurityPrivateKey)
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

func (t *UDP) WriteSecurity() error {
	if config.SecurityPublicKey == nil || len(config.SecurityPublicKey) <= 0 {
		return fmt.Errorf("no public key")
	}

	key, iv := utils.GenerateAES()
	t.AesKey = key
	t.AesIv = iv

	keyIv := make([]byte, len(key)+len(iv))
	copy(keyIv[0:len(key)], key)
	copy(keyIv[len(key):], iv)
	messageBuf, err := utils.EncryptRSA(keyIv, config.SecurityPublicKey)
	if err != nil {
		return err
	}
	err = t.WriteMessage(&msg.SecurityRequest{KeyIv: messageBuf}, nil)
	if err != nil {
		return err
	}
	return nil
}

func (t *UDP) WriteMessage(message msg.Message, addr *net.UDPAddr) error {
	return t.writeMessage(message, addr, false)
}

func (t *UDP) WriteSecurityMessage(message msg.Message, addr *net.UDPAddr) error {
	return t.writeMessage(message, addr, true)
}

func (t *UDP) writeMessage(message msg.Message, addr *net.UDPAddr, security bool) error {
	t.writeLock.Lock()
	defer t.writeLock.Unlock()

	// UDPPackage 特殊处理
	switch m := message.(type) {
	case *msg.UDPPackage:
		return t.writeUDPPackage(m, addr)
	}

	// 解析
	data, err := msg.ToByte(message)
	if err != nil {
		return err
	}
	if len(data) <= 0 {
		return nil
	}

	// 加密
	if security && t.Security && config.NewConfig.Common.Security.Enable {
		if data, err = utils.AesEncrypt(data, t.AesKey, t.AesIv); err != nil {
			return err
		}
	}

	// 整合包
	buf := make([]byte, len(data)+4)
	copy(buf[:4], utils.I2b32(message.GetMessageType()))
	copy(buf[4:], data)

	if len(buf) > maxPackageLen {
		log.Warn("udp package len", len(buf))
	}

	// 发送消息
	if addr == nil {
		if _, err = t.UDPConn.Write(buf); err != nil {
			return err
		}
	} else {
		if _, err = t.UDPConn.WriteToUDP(buf, addr); err != nil {
			return err
		}
	}
	return nil
}
func (t *UDP) writeUDPPackage(message *msg.UDPPackage, addr *net.UDPAddr) error {
	messageId := utils.GetId()
	messageLen := len(message.D)
	tb := utils.I2b32(message.GetMessageType()) // 消息类型
	mb := utils.I2b64(messageId)                // 消息id
	lb := utils.I2b32(uint32(messageLen))       // 消息长度
	createWriteUDP(messageId)

	o := 0 // 起始位置偏移量
	for o < messageLen {
		d := message.D[o:]
		if len(d) > maxPackageDataLen {
			d = d[:maxPackageDataLen]
		}
		// 发送消息
		buf := make([]byte, len(d)+20)
		copy(buf[:4], tb)
		copy(buf[4:12], mb)
		copy(buf[12:16], lb)
		copy(buf[16:20], utils.I2b32(uint32(o)))
		buf[20] = uint8(message.T)
		copy(buf[21:], d)
		addWriteUDP(messageId, uint32(o), t.UDPConn, addr, buf)
		o += len(d)
	}
	go reWriteUDP(messageId)
	return nil
}

func createWriteUDP(messageId utils.ID) {
	writeUDPCacheLock.Lock()
	defer writeUDPCacheLock.Unlock()
	cache := make(map[uint32]*writeUDPCache)
	writeUDPCacheMap[messageId] = cache
}

func addWriteUDP(messageId utils.ID, o uint32, conn *net.UDPConn, addr *net.UDPAddr, buf []byte) {
	writeUDPCacheLock.Lock()
	defer writeUDPCacheLock.Unlock()
	writeUDPCacheMap[messageId][o] = &writeUDPCache{conn, addr, buf}
}
func reWriteUDP(messageId utils.ID) {
	for i := 0; i < 40; i++ {
		if s := doReWriteUDP(messageId); s {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}
func doReWriteUDP(messageId utils.ID) bool {
	writeUDPCacheLock.Lock()
	defer writeUDPCacheLock.Unlock()
	cache, ok := writeUDPCacheMap[messageId]
	if !ok {
		return true
	}
	for _, writeUDPCache := range cache {
		if writeUDPCache.addr == nil {
			_, _ = writeUDPCache.conn.Write(writeUDPCache.buf)
		} else {
			_, _ = writeUDPCache.conn.WriteToUDP(writeUDPCache.buf, writeUDPCache.addr)
		}
	}
	return false
}
func RemoteWriteUDP(messageId utils.ID, o uint32) {
	writeUDPCacheLock.Lock()
	defer writeUDPCacheLock.Unlock()
	if cache, ok := writeUDPCacheMap[messageId]; ok {
		if _, ok = cache[o]; ok {
			delete(cache, o)
			if len(cache) == 0 {
				delete(writeUDPCacheMap, messageId)
				utils.RemoteId(messageId)
			}
		}
	}
}

func (t *UDP) ReadMessage(timeout time.Time) (message Message, addr *net.UDPAddr, err error) {
	return t.readMessage(timeout, false)
}

func (t *UDP) ReadSecurityMessage(timeout time.Time) (message Message, addr *net.UDPAddr, err error) {
	return t.readMessage(timeout, true)
}

func (t *UDP) readMessage(timeout time.Time, security bool) (message Message, addr *net.UDPAddr, err error) {
	t.readLock.Lock()
	defer t.readLock.Unlock()

	_ = t.SetReadDeadline(timeout)

	var readLength int
	var needContinue = true
	for needContinue {
		readLength, addr, err = t.UDPConn.ReadFromUDP(t.Buf)
		if err != nil {
			return
		}
		message, needContinue = t.ToObj(t.Buf[:readLength], addr, security)
	}
	_ = t.SetReadDeadline(time.Time{})
	return
}
func (t *UDP) readMessageNoAddr(timeout time.Time, security bool) (message Message, err error) {
	t.readLock.Lock()
	defer t.readLock.Unlock()

	_ = t.SetReadDeadline(timeout)

	var readLength int
	var needContinue = true
	for needContinue {
		readLength, err = t.UDPConn.Read(t.Buf)
		if err != nil {
			return
		}
		message, needContinue = t.ToObj(t.Buf[:readLength], nil, security)
	}
	_ = t.SetReadDeadline(time.Time{})
	return
}

func (t *UDP) ToObj(buf []byte, addr *net.UDPAddr, security bool) (message Message, needContinue bool) {
	if len(buf) < 4 {
		message = Message{Err: fmt.Errorf("udp package len < 4, %d", len(buf))}
		return
	}
	data := buf[4:]

	// 获取消息类型
	messageType, err := utils.B2i32(buf[:4])
	if err != nil {
		message = Message{Err: err}
		return
	}
	m, err := msg.NewMessage(messageType)
	if err != nil {
		message = Message{Err: err}
		return
	}

	// UDPPackage 特殊处理
	switch m.(type) {
	case *msg.UDPPackage:
		return t.readUDPPackage(data, addr)
	}

	// 解密
	if security && t.Security && config.NewConfig.Common.Security.Enable {
		if data, err = utils.AesDecrypt(data, t.AesKey, t.AesIv); err != nil {
			message = Message{Err: err}
			return
		}
	}

	// 解析
	if err = msg.ToObj(data, m); err != nil {
		message = Message{Err: err}
		return
	}

	message = Message{Message: m, Err: nil}
	return
}
func (t *UDP) readUDPPackage(buf []byte, addr *net.UDPAddr) (result Message, needContinue bool) {
	if len(buf) < 17 {
		result = Message{Err: fmt.Errorf("udp package len < 16, %d", len(buf))}
		return
	}
	needContinue = true
	m, _ := utils.B2i64(buf[:8])    // 消息id
	l, _ := utils.B2i32(buf[8:12])  // 消息长度
	o, _ := utils.B2i32(buf[12:16]) // 起始位置偏移量
	typ := msg.ClientType(buf[16])
	d := buf[17:] // 数据
	udpPackageCacheLock.Lock()
	defer udpPackageCacheLock.Unlock()
	// 返回成功消息
	_ = t.WriteMessage(&msg.UDPPackageConfirm{M: m, O: o}, addr)

	message, ok := udpPackageCache[m]
	if !ok {
		message = &msg.UDPPackage{M: m, L: l, O: 0, S: 0, T: typ, D: make([]byte, int(l))}
		message.OMap = make(map[uint32]uint32)
		udpPackageCache[m] = message
	}
	// 去重
	if message.Stop {
		return
	}
	if _, ok = message.OMap[o]; !ok {
		message.OMap[o] = o
		copy(message.D[int(o):], d)
		message.S += uint32(len(d))
		if message.S == l {
			message.Stop = true
			return Message{Message: message}, false
		}
	}
	message.LastTime = time.Now()
	return
}

func (t *UDP) ServerInit(message msg.Message, addr *net.UDPAddr) (do bool, err error) {
	do = true
	switch m := message.(type) {
	case *msg.NewUDP:
		// TODO 没有释放Id
		t.ServerId = utils.GetId()
		t.Status = "new"
		// 设置协议
		_ = t.WriteMessage(&msg.SrpRequest{
			Project:  "srp",
			Version:  config.Version,
			Security: config.NewConfig.Common.Security.Enable,
		}, addr)
	case *msg.SecurityRequest:
		if err = t.ReadSecurity(m); err != nil {
			err = fmt.Errorf("security error: %s", err)
			_ = t.WriteMessage(&msg.SecurityResponse{Result: err.Error()}, addr)
		} else {
			_ = t.WriteMessage(&msg.SecurityResponse{Result: "success"}, addr)
			t.Security = true
		}
	case *msg.InfoRequest:
		if err = config.VerifyVersion(m.Version); err != nil {
			_ = t.WriteSecurityMessage(&msg.InfoResponse{
				Version:  config.Version,
				ServerId: t.ServerId,
				Result:   err.Error(),
			}, addr)
			return
		}
		t.ClientName = m.ClientName
		_ = t.WriteSecurityMessage(&msg.InfoResponse{Version: config.Version, ServerId: t.ServerId, Result: "success"}, addr)
		t.Status = "Run"
	default:
		do = false
	}
	return
}

func (t *UDP) ClientInit(connectType msg.ClientType) error {
	t.Status = "new"
	// 获取协议
	_ = t.WriteMessage(&msg.NewUDP{Id: 13520}, nil)
	message, _, err := t.ReadMessage(time.Now().Add(8 * time.Second))
	if err != nil {
		return fmt.Errorf("read srp error1: %s", err)
	}
	if message.Err != nil {
		return fmt.Errorf("read srp error2: %s", message.Err)
	}
	srpRequest, convertResult := message.Message.(*msg.SrpRequest)
	if !convertResult {
		return fmt.Errorf("read srp type error: %d", message.Message.GetMessageType())
	}

	// 校验版本
	if err := config.VerifyVersion(srpRequest.Version); err != nil {
		return fmt.Errorf("verify version error: %s", err.Error())
	}
	if srpRequest.Security != config.NewConfig.Common.Security.Enable {
		return fmt.Errorf("security error, client: %v, server: %v", config.NewConfig.Common.Security.Enable, srpRequest.Security)
	}

	// 设置加密
	if config.NewConfig.Common.Security.Enable {
		t.Status = "Start Write Security"
		if err := t.WriteSecurity(); err != nil {
			return fmt.Errorf("write Security error: %s", err)
		}
		message, _, err = t.ReadMessage(time.Now().Add(8 * time.Second))
		if err != nil {
			return fmt.Errorf("security error1: %s", err)
		}
		if message.Err != nil {
			return fmt.Errorf("security error2: %s", message.Err)
		}
		securityResponse, convertResult := message.Message.(*msg.SecurityResponse)
		if !convertResult {
			return fmt.Errorf("read info type error: %d", message.Message.GetMessageType())
		}

		if securityResponse.Result != "success" {
			return fmt.Errorf("security error: %s", securityResponse.Result)
		}
		t.Security = true
	}

	// 设置基础信息
	t.Status = "Write Info"
	t.ClientName = config.NewConfig.Common.Client.Name
	_ = t.WriteSecurityMessage(&msg.InfoRequest{
		ClientName:  t.ClientName,
		Version:     config.Version,
		ConnectType: connectType,
	}, nil)

	message, _, err = t.ReadSecurityMessage(time.Now().Add(8 * time.Second))
	if err != nil {
		return fmt.Errorf("read info error: %s", err)
	}
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
	t.Status = "Run"
	return nil
}

func (t *UDP) TransferR(dest *net.UDPConn, finish chan interface{}) {
	t.transferR(dest, nil, finish)
}

func (t *UDP) TransferRByUDP(dest *net.UDPConn, clientAddr *net.UDPAddr, finish chan interface{}) {
	t.transferR(dest, clientAddr, finish)
}

func (t *UDP) transferR(dest *net.UDPConn, clientAddr *net.UDPAddr, finish chan interface{}) {
	defer func() {
		_ = t.Close()
		_ = dest.Close()
	}()
	for {
		message, err := t.readMessageNoAddr(time.Time{}, false)
		if t.TimeOut {
			break
		}
		if err != nil {
			break
		}
		if message.Err != nil {
			log.Debug("read udp package error", message.Err)
			return
		}
		switch udpPackage := message.Message.(type) {
		case *msg.UDPPackage:
			if clientAddr != nil {
				_, _ = dest.WriteToUDP(udpPackage.D, clientAddr)
			} else {
				_, _ = dest.Write(udpPackage.D)
			}
			t.LastTransferTime = time.Now()
		case *msg.UDPPackageConfirm:
			RemoteWriteUDP(udpPackage.M, udpPackage.O)
		default:
			log.Debug("read udp package type error", message.Message.GetMessageType())
			return
		}
	}

	// 如果有丢包的话试一下替换下面的代码
	//buf := make([]byte, 64*1024)
	//for {
	//	length, err := t.Read(buf)
	//	if t.TimeOut {
	//		break
	//	}
	//	if err != nil {
	//		break
	//	}
	//
	//	buffer := make([]byte, length)
	//	copy(buffer, buf[:length])
	//
	//	go func(buffer []byte) {
	//		t.LastTransferTime = time.Now()
	//
	//		message, needContinue := t.ToObj(buffer, nil, false)
	//		if needContinue {
	//			return
	//		}
	//		if message.Err != nil {
	//			log.Debug("read udp package error", message.Err)
	//			return
	//		}
	//		switch udpPackage := message.Message.(type) {
	//		case *msg.UDPPackage:
	//			_, _ = dest.Write(udpPackage.D)
	//		case *msg.UDPPackageConfirm:
	//			RemoteWriteUDP(udpPackage.M, udpPackage.O)
	//		default:
	//			log.Debug("read udp package type error", message.Message.GetMessageType())
	//			return
	//		}
	//	}(buffer)
	//}

	finish <- 1
}

func (t *UDP) TransferW(src *net.UDPConn, finish chan interface{}, typ msg.ClientType) {
	defer func() {
		_ = src.Close()
		_ = t.Close()
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

		buffer := make([]byte, length)
		copy(buffer, buf[:length])
		_ = t.WriteMessage(&msg.UDPPackage{T: typ, D: buffer}, nil)
		t.LastTransferTime = time.Now()
	}

	finish <- 1
}

func (t *UDP) GoTimeOut(closeFun func()) {
	time.Sleep(timeout)
	for {
		if time.Now().Sub(t.LastTransferTime) > timeout {
			break
		}
		time.Sleep(t.LastTransferTime.Add(timeout).Sub(time.Now()))
	}
	t.TimeOut = true
	closeFun()
}
