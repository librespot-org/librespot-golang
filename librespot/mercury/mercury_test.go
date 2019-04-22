package mercury

import (
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"github.com/librespot-org/librespot-golang/Spotify"
	"github.com/librespot-org/librespot-golang/librespot/connection"
	"github.com/librespot-org/librespot-golang/librespot/core"
	"github.com/librespot-org/librespot-golang/librespot/spirc"
	"testing"
)

func setupTestController(stream connection.PacketStream) *spirc.Controller {
	s := &core.Session{
		stream:   stream,
		deviceId: "testDevice",
	}
	s.mercury = CreateMercury(s)
	return setupController(s, "fakeUser", []byte{})
}

func TestMultiPart(t *testing.T) {
	stream := &fakeStream{
		recvPackets: make(chan shanPacket, 5),
		sendPackets: make(chan shanPacket, 5),
	}

	controller := setupTestController(stream)

	subHeader := &Spotify.Header{
		Uri: proto.String("hm://searchview/km/v2/search/Future"),
	}
	subHeaderData, _ := proto.Marshal(subHeader)

	header := &Spotify.Header{
		Uri:         proto.String("hm://searchview/km/v2/search/Future"),
		ContentType: proto.String("application/json; charset=UTF-8"),
		StatusCode:  proto.Int32(200),
	}
	body := []byte("{searchResults: {tracks: [], albums: [], tracks: []}}")

	headerData, _ := proto.Marshal(header)
	seq := []byte{0, 0, 0, 1}

	p0, _ := encodeMercuryHead([]byte{0, 0, 0, 0}, 1, 1)
	binary.Write(p0, binary.BigEndian, uint16(len(subHeaderData)))
	p0.Write(subHeaderData)

	p1, _ := encodeMercuryHead(seq, 1, 0)
	binary.Write(p1, binary.BigEndian, uint16(len(headerData)))
	p1.Write(headerData)

	p2, _ := encodeMercuryHead(seq, 1, 1)
	binary.Write(p2, binary.BigEndian, uint16(len(body)))
	p2.Write(body)

	didRecieveCallback := false
	controller.session.mercurySendRequest(Request{
		Method:  "SEND",
		Uri:     "hm://searchview/km/v2/search/Future",
		Payload: [][]byte{},
	}, func(res Response) {
		didRecieveCallback = true
		if string(res.Payload[0]) != string(body) {
			t.Errorf("bad body received")
		}
	})

	stream.recvPackets <- shanPacket{cmd: 0xb2, buf: p0.Bytes()}
	stream.recvPackets <- shanPacket{cmd: 0xb2, buf: p1.Bytes()}
	stream.recvPackets <- shanPacket{cmd: 0xb2, buf: p2.Bytes()}

	controller.session.poll()
	controller.session.poll()
	controller.session.poll()

	if !didRecieveCallback {
		t.Errorf("never received callback")
	}

}
