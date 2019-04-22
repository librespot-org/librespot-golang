package connection

import (
	"encoding/binary"
	"io"
	"sync"
)

// PlainConnection represents an unencrypted connection to a Spotify AP
type PlainConnection struct {
	Writer io.Writer
	Reader io.Reader
	mutex  *sync.Mutex
}

func makePacketPrefix(prefix []byte, data []byte) []byte {
	size := len(prefix) + 4 + len(data)
	buf := make([]byte, 0, size)
	buf = append(buf, prefix...)
	sizeBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(sizeBuf, uint32(size))
	buf = append(buf, sizeBuf...)
	return append(buf, data...)
}

func MakePlainConnection(reader io.Reader, writer io.Writer) PlainConnection {
	return PlainConnection{
		Reader: reader,
		Writer: writer,
		mutex:  &sync.Mutex{},
	}
}

func (p *PlainConnection) SendPrefixPacket(prefix []byte, data []byte) (packet []byte, err error) {
	packet = makePacketPrefix(prefix, data)

	p.mutex.Lock()
	_, err = p.Writer.Write(packet)
	p.mutex.Unlock()

	return
}

func (p *PlainConnection) RecvPacket() (buf []byte, err error) {
	var size uint32
	err = binary.Read(p.Reader, binary.BigEndian, &size)
	if err != nil {
		return
	}
	buf = make([]byte, size)
	binary.BigEndian.PutUint32(buf, size)
	_, err = io.ReadFull(p.Reader, buf[4:])
	if err != nil {
		return
	}
	return buf, nil
}
