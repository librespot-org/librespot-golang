package spotcontrol

import (
	"encoding/binary"
	"io"
)

type plainConnection struct {
	writer io.Writer
	reader io.Reader
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

func makePlainConnection(reader io.Reader, writer io.Writer) plainConnection {
	return plainConnection{
		reader: reader,
		writer: writer,
	}
}

func (p *plainConnection) SendPrefixPacket(prefix []byte, data []byte) (packet []byte, err error) {
	packet = makePacketPrefix(prefix, data)
	_, err = p.writer.Write(packet)
	return
}

func (p *plainConnection) RecvPacket() (buf []byte, err error) {
	var size uint32
	err = binary.Read(p.reader, binary.BigEndian, &size)
	if err != nil {
		return
	}
	buf = make([]byte, size)
	binary.BigEndian.PutUint32(buf, size)
	_, err = io.ReadFull(p.reader, buf[4:])
	if err != nil {
		return
	}
	return buf, nil
}
