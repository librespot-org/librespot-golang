package crypto

import (
	"bytes"
	"encoding/binary"
	"github.com/librespot-org/librespot-golang/librespot/connection"
	"io"
	"log"
	"sync"
)

type shannonStream struct {
	sendNonce  uint32
	sendCipher shn_ctx
	recvCipher shn_ctx

	recvNonce uint32
	reader    io.Reader
	writer    io.Writer

	mutex *sync.Mutex
}

func setKey(ctx *shn_ctx, key []uint8) {
	shn_key(ctx, key, len(key))

	nonce := make([]byte, 4)
	binary.BigEndian.PutUint32(nonce, 0)
	shn_nonce(ctx, nonce, len(nonce))
}

// CreateStream initializes a new Shannon-encrypted PacketStream connection from the specified keys and plain connection
func CreateStream(keys SharedKeys, conn connection.PlainConnection) connection.PacketStream {
	s := &shannonStream{
		reader: conn.Reader,
		writer: conn.Writer,
		mutex:  &sync.Mutex{},
	}

	setKey(&s.recvCipher, keys.recvKey)
	setKey(&s.sendCipher, keys.sendKey)

	return s
}

func (s *shannonStream) SendPacket(cmd uint8, data []byte) (err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

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
	return s.EncryptBytes(messageBytes)
}

func (s *shannonStream) EncryptBytes(messageBytes []byte) []byte {
	shn_encrypt(&s.sendCipher, messageBytes, len(messageBytes))
	return messageBytes
}

func (s *shannonStream) Decrypt(messageBytes []byte) []byte {
	shn_decrypt(&s.recvCipher, messageBytes, len(messageBytes))
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
	shn_finish(&s.sendCipher, mac, count)

	s.sendNonce += 1
	nonce := make([]uint8, 4)
	binary.BigEndian.PutUint32(nonce, s.sendNonce)
	shn_nonce(&s.sendCipher, nonce, len(nonce))

	_, err = s.writer.Write(mac)
	return
}

func (s *shannonStream) finishRecv() {
	count := 4

	mac := make([]byte, count)
	io.ReadFull(s.reader, mac)

	mac2 := make([]byte, count)
	shn_finish(&s.recvCipher, mac2, count)

	if !bytes.Equal(mac, mac2) {
		log.Println("received mac doesn't match")
	}

	s.recvNonce += 1
	nonce := make([]uint8, 4)
	binary.BigEndian.PutUint32(nonce, s.recvNonce)
	shn_nonce(&s.recvCipher, nonce, len(nonce))
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

	return cmd, buf, err
}
