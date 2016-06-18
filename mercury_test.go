package spotcontrol

import (
	"encoding/binary"
	Spotify "github.com/badfortrains/spotcontrol/proto"
	"github.com/golang/protobuf/proto"
	"testing"
)

func setupTestController(stream packetStream) *SpircController {
	s := &session{
		stream:   stream,
		deviceId: "testDevice",
	}
	s.mercury = setupMercury(s)
	return setupController(s, "fakeUser")
}

func TestMultiPart(t *testing.T) {
	stream := &fakeStream{
		recvPackets: make(chan shanPacket, 5),
		sendPackets: make(chan shanPacket, 5),
	}

	controller := setupTestController(stream)

	header := &Spotify.Header{
		Uri:         proto.String("hm://searchview/km/v2/search/Future"),
		ContentType: proto.String("application/json; charset=UTF-8"),
		StatusCode:  proto.Int32(200),
	}
	body := []byte("{searchResults: {tracks: [], albums: [], tracks: []}}")

	headerData, _ := proto.Marshal(header)
	seq := []byte{0, 0, 0, 2}

	p1, _ := encodeMercuryHead(seq, 1, 0)
	binary.Write(p1, binary.BigEndian, uint16(len(headerData)))
	p1.Write(headerData)

	p2, _ := encodeMercuryHead(seq, 1, 1)
	binary.Write(p2, binary.BigEndian, uint16(len(body)))
	p2.Write(body)

	didRecieveCallback := false
	controller.session.mercurySendRequest(mercuryRequest{
		method:  "SEND",
		uri:     "hm://searchview/km/v2/search/Future",
		payload: [][]byte{},
	}, func(res mercuryResponse) {
		didRecieveCallback = true
		if string(res.payload[0]) != string(body) {
			t.Errorf("bad body received")
		}
	})

	stream.recvPackets <- shanPacket{cmd: 0xb2, buf: p1.Bytes()}
	stream.recvPackets <- shanPacket{cmd: 0xb2, buf: p2.Bytes()}

	controller.session.poll()
	controller.session.poll()

	if !didRecieveCallback {
		t.Errorf("never received callback")
	}

}
