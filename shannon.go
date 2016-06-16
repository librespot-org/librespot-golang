package spotcontrol

// #include "./shn.h"
import "C"

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"unsafe"
)

type shnCtx C.shn_ctx

type shannonStream struct {
	sendNonce  uint32
	sendCipher C.shn_ctx

	recvNonce  uint32
	recvCipher C.shn_ctx
	reader     io.Reader
	writer     io.Writer
}

func setKey(ctx *C.shn_ctx, key []uint8) {
	C.shn_key(ctx,
		(*C.uchar)(unsafe.Pointer(&key[0])),
		C.int(len(key)))

	nonce := make([]byte, 4)
	binary.BigEndian.PutUint32(nonce, 0)
	C.shn_nonce(ctx,
		(*C.uchar)(unsafe.Pointer(&nonce[0])),
		C.int(len(nonce)))
}

func setupStream(keys sharedKeys, conn plainConnection) packetStream {
	s := &shannonStream{
		reader: conn.reader,
		writer: conn.writer,
	}

	setKey(&s.recvCipher, keys.recvKey)
	setKey(&s.sendCipher, keys.sendKey)
	return s
}

func (s *shannonStream) SendPacket(cmd uint8, data []byte) (err error) {
	_, err = s.Write(cipherPacket(cmd, data))
	if err != nil {
		return
	}
	err = s.FinishSend()
	return
}

func cipherPacket(cmd uint8, data []byte) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, cmd)
	binary.Write(buf, binary.BigEndian, uint16(len(data)))
	buf.Write(data)
	return buf.Bytes()
}

func (s *shannonStream) Encrypt(message string) []byte {
	messageBytes := []byte(message)
	C.shn_encrypt(&s.sendCipher,
		(*C.uchar)(unsafe.Pointer(&messageBytes[0])),
		C.int(len(messageBytes)))

	return messageBytes
}

func (s *shannonStream) EncryptBytes(messageBytes []byte) []byte {
	C.shn_encrypt(&s.sendCipher,
		(*C.uchar)(unsafe.Pointer(&messageBytes[0])),
		C.int(len(messageBytes)))

	return messageBytes
}

func (s *shannonStream) Decrypt(messageBytes []byte) []byte {
	C.shn_decrypt(&s.recvCipher,
		(*C.uchar)(unsafe.Pointer(&messageBytes[0])),
		C.int(len(messageBytes)))

	return messageBytes
}

func (s *shannonStream) WrapReader(reader io.Reader) {
	s.reader = reader
}

func (s *shannonStream) WrapWriter(writer io.Writer) {
	s.writer = writer
}

func (s *shannonStream) Read(p []byte) (n int, err error) {
	n, err = s.reader.Read(p)
	p = s.Decrypt(p)
	return n, err
}

func (s *shannonStream) Write(p []byte) (n int, err error) {
	p = s.EncryptBytes(p)
	return s.writer.Write(p)
}

func (s *shannonStream) FinishSend() (err error) {
	count := 4
	mac := make([]byte, count)
	C.shn_finish(&s.sendCipher,
		(*C.uchar)(unsafe.Pointer(&mac[0])),
		C.int(count))

	s.sendNonce += 1
	nonce := make([]uint8, 4)
	binary.BigEndian.PutUint32(nonce, s.sendNonce)
	C.shn_nonce(&s.sendCipher,
		(*C.uchar)(unsafe.Pointer(&nonce[0])),
		C.int(len(nonce)))

	_, err = s.writer.Write(mac)
	return
}

func (s *shannonStream) finishRecv() {
	count := 4

	mac := make([]byte, count)
	io.ReadFull(s.reader, mac)

	mac2 := make([]byte, count)
	C.shn_finish(&s.recvCipher,
		(*C.uchar)(unsafe.Pointer(&mac2[0])),
		C.int(count))

	if !bytes.Equal(mac, mac2) {
		//log.Fatal("received mac doesn't match")
		log.Println("received mac doesn't match")
	}

	s.recvNonce += 1
	nonce := make([]uint8, 4)
	binary.BigEndian.PutUint32(nonce, s.recvNonce)
	C.shn_nonce(&s.recvCipher,
		(*C.uchar)(unsafe.Pointer(&nonce[0])),
		C.int(len(nonce)))
}

func (s *shannonStream) RecvPacket() (cmd uint8, buf []byte, err error) {
	err = binary.Read(s, binary.BigEndian, &cmd)
	if err != nil {
		return
	}
	var size uint16
	err = binary.Read(s, binary.BigEndian, &size)
	if err != nil {
		return
	}
	if size > 0 {
		buf = make([]byte, size)
		_, err = io.ReadFull(s.reader, buf)
		if err != nil {
			return
		}
		buf = s.Decrypt(buf)

	}
	s.finishRecv()

	return
}
