package spirc

import (
	"bytes"
	"github.com/golang/protobuf/proto"
	"github.com/librespot-org/librespot-golang/Spotify"
	"github.com/librespot-org/librespot-golang/librespot/connection"
	"github.com/librespot-org/librespot-golang/librespot/mercury"
	"testing"
)

type fakeServer struct {
	stream    connection.PacketStream
	responses []mercury.Response
	mInternal *mercury.Internal
}

func setupFakeServer(stream connection.PacketStream) *fakeServer {
	return &fakeServer{
		stream:    stream,
		responses: make([]mercury.Response, 5),
		mInternal: &mercury.Internal{
			pending: make(map[string]mercury.Pending),
			stream:  stream,
		},
	}
}

func (f *fakeServer) getResponse(t *testing.T) *mercury.Response {
	cmd, data, err := f.stream.RecvPacket()
	if err != nil {
		t.Error("poll error", err)
	}
	if 0xb2 <= cmd && cmd <= 0xb6 || cmd == 0x1b {
		var response *mercuryResponse
		var err error
		for ; response == nil && err == nil; response, err = f.mInternal.parseResponse(cmd, bytes.NewReader(data)) {
			if err != nil {
				t.Errorf("Handle 0xbx %q ", err)
			}
		}
		return response
	} else {
		t.Errorf("Non mercury command %q ", cmd)
	}
	return nil
}

func (f *fakeServer) getResponseFrame(t *testing.T) (*Spotify.Frame, *mercuryResponse) {
	response := f.getResponse(t)
	frame := &Spotify.Frame{}
	proto.Unmarshal(response.headerData, frame)
	return frame, response
}

func setupContollerAndServer(t *testing.T) (*Controller, *fakeServer) {
	sessionStream := &fakeStream{
		recvPackets: make(chan shanPacket, 5),
		sendPackets: make(chan shanPacket, 5),
	}

	serverStream := &fakeStream{
		recvPackets: sessionStream.sendPackets,
		sendPackets: sessionStream.recvPackets,
	}

	server := setupFakeServer(serverStream)

	s := &session{
		stream:   sessionStream,
		deviceId: "testDevice",
	}
	s.mercury = setupMercury(s)
	controller := CreateController(s, "fakeUser", []byte{})
	return controller, server
}

func TestHelloCmd(t *testing.T) {
	controller, server := setupContollerAndServer(t)
	go controller.SendHello()

	frame, response := server.getResponseFrame(t)
	if response.uri != "hm://remote/user/fakeUser/" {
		t.Errorf("Bad response uri %q ", response.uri)
	}
	if frame.GetTyp() != Spotify.MessageType_kMessageTypeHello {
		t.Errorf("Wrong message type")
	}
}
