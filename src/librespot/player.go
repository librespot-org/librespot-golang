package librespot

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type Player struct {
	session session
	seq     uint32
}

func (p *Player) loadTrack(trackId string, fileId string) error {
	// Request audio key
	cmd, data, err := p.session.SendAndRecvStreamPacket(0xc, p.buildKeyRequest(trackId, fileId))

	fmt.Printf("CMD: %v", cmd)
	fmt.Printf("DATA: %v", data)
	fmt.Printf("ERR: %v", err)

	return err
}

func (p *Player) buildKeyRequest(trackId string, fileId string) []byte {
	fourBytes := make([]byte, 4)
	twoBytes := make([]byte, 2)

	bs := make([]byte, len(fileId)+len(trackId)+4+2)
	buf := bytes.NewBuffer(bs)

	buf.WriteString(fileId)
	buf.WriteString(trackId)

	binary.BigEndian.PutUint32(fourBytes, p.seq)
	buf.Write(fourBytes)
	p.seq++

	binary.BigEndian.PutUint16(twoBytes, 0x0000)
	buf.Write(twoBytes)

	return bs
}
