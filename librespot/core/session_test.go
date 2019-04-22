package core

import (
	"bytes"
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"github.com/librespot-org/librespot-golang/Spotify"
	"github.com/librespot-org/librespot-golang/librespot/connection"
	"github.com/librespot-org/librespot-golang/librespot/crypto"
	"github.com/librespot-org/librespot-golang/librespot/mercury"
	"github.com/librespot-org/librespot-golang/librespot/spirc"
	"io"
	"math/big"
	"testing"
)

type shanPacket struct {
	cmd uint8
	buf []byte
}

type fakeStream struct {
	recvPackets chan shanPacket
	sendPackets chan shanPacket
}

func (f *fakeStream) SendPacket(cmd uint8, data []byte) (err error) {
	f.sendPackets <- shanPacket{cmd: cmd, buf: data}
	return nil
}

func (f *fakeStream) RecvPacket() (cmd uint8, buf []byte, err error) {
	p := <-f.recvPackets
	return p.cmd, p.buf, nil
}

func readPlainPart(reader io.Reader, prefixSize uint32) ([]byte, error) {
	if prefixSize > 0 {
		prefix := make([]byte, prefixSize)
		_, _ = io.ReadFull(reader, prefix)
	}

	var size uint32
	binary.Read(reader, binary.BigEndian, &size)
	buf := make([]byte, size-4-prefixSize)
	_, err := io.ReadFull(reader, buf)
	return buf, err
}

func checkHead(t *testing.T, buf io.Reader) {
	handleHead(buf)
	headerData, _ := parsePart(buf)
	header := &Spotify.Header{}
	proto.Unmarshal(headerData, header)

	if *header.Uri != "hm://remote/user/fakeUser/" {
		t.Errorf("Wrong username  Got %q, ", header.Uri)
	}

	if *header.Method != "SEND" {
		t.Errorf("Wrong method")
	}
}

type fakeCon struct {
	reader *bytes.Buffer
	writer *bytes.Buffer
}

func (f *fakeCon) Write(b []byte) (n int, err error) {
	return f.writer.Write(b)
}

func (f *fakeCon) Read(b []byte) (n int, err error) {
	return f.reader.Read(b)
}

func TestLogin(t *testing.T) {
	conn := &fakeCon{
		reader: bytes.NewBuffer(make([]byte, 0)),
		writer: bytes.NewBuffer(make([]byte, 0)),
	}

	fakeShan := &fakeStream{
		recvPackets: make(chan shanPacket),
		sendPackets: make(chan shanPacket),
	}

	s := &Session{
		deviceId:           "testDevice",
		keys:               crypto.GenerateKeysFromPrivate(big.NewInt(20.0), make([]byte, 10)),
		tcpCon:             conn,
		shannonConstructor: func(keys crypto.SharedKeys, conn connection.PlainConnection) connection.PacketStream { return fakeShan },
		mercuryConstructor: mercury.CreateMercury,
	}

	serverResponse := &Spotify.APResponseMessage{
		Challenge: &Spotify.APChallenge{
			LoginCryptoChallenge: &Spotify.LoginCryptoChallengeUnion{
				DiffieHellman: &Spotify.LoginCryptoDiffieHellmanChallenge{
					Gs:                 []byte{25},
					ServerSignatureKey: proto.Int32(5),
					GsSignature:        []byte{5},
				},
			},
			FingerprintChallenge: &Spotify.FingerprintChallengeUnion{},
			PowChallenge:         &Spotify.PoWChallengeUnion{},
			CryptoChallenge:      &Spotify.CryptoChallengeUnion{},
			ServerNonce:          []byte{5},
		},
	}

	serverResponseData, _ := proto.Marshal(serverResponse)
	binary.Write(conn.reader, binary.BigEndian, uint32(len(serverResponseData)+4))
	//Write initial server response to plain connection
	conn.reader.Write(serverResponseData)

	result := make(chan []byte, 2)
	go func() {
		err := s.loginSession("testUser", "123", "myDevice")
		if err != nil {
			t.Errorf("bad return values")
		}
		result <- s.reusableAuthBlob
	}()

	//Get the login packet sent to the spotify server from spotcontrol
	loginPacket := <-fakeShan.sendPackets
	clientResponse := &Spotify.ClientResponseEncrypted{}
	proto.Unmarshal(loginPacket.buf, clientResponse)

	if *clientResponse.LoginCredentials.Username != "testUser" {
		t.Errorf("bad auth user")
	}
	if !bytes.Equal(clientResponse.LoginCredentials.AuthData, []byte("123")) {
		t.Errorf("bad auth password")
	}

	plainClientRes := &Spotify.ClientResponsePlaintext{}
	// Discard original hello message
	readPlainPart(conn.writer, 2)
	// Get plain client response from plain connection
	plainData, _ := readPlainPart(conn.writer, 0)
	proto.Unmarshal(plainData, plainClientRes)
	hmac := []byte{226, 239, 29, 188, 200, 160, 193, 245, 71, 39, 15, 82, 156, 34, 168, 224, 134, 149, 128, 222}
	if !bytes.Equal(plainClientRes.LoginCryptoResponse.DiffieHellman.Hmac, hmac) {
		t.Errorf("failed hmac comparison", plainClientRes.LoginCryptoResponse.DiffieHellman.Hmac)
	}

	welcome := &Spotify.APWelcome{
		CanonicalUsername:           proto.String("testUserCanonical"),
		AccountTypeLoggedIn:         Spotify.AccountType_Spotify.Enum(),
		CredentialsTypeLoggedIn:     Spotify.AccountType_Spotify.Enum(),
		ReusableAuthCredentialsType: Spotify.AuthenticationType_AUTHENTICATION_USER_PASS.Enum(),
		ReusableAuthCredentials:     []byte{0, 1, 2},
	}
	welcomeData, _ := proto.Marshal(welcome)

	fakeShan.recvPackets <- shanPacket{cmd: 0xac, buf: welcomeData}
	// country code
	fakeShan.recvPackets <- shanPacket{cmd: 0x1b, buf: []byte{0, 1}}
	// ignore subscribe
	<-fakeShan.sendPackets
	welcomeRes := <-result
	if !bytes.Equal(welcomeRes, []byte{0, 1, 2}) {
		t.Errorf("Wrong authdata returned.  Got %v", welcomeRes)
	}
}

func TestHello(t *testing.T) {
	stream := fakeStream{
		recvPackets: make(chan shanPacket),
		sendPackets: make(chan shanPacket, 2),
	}

	s := &Session{
		stream:   &stream,
		deviceId: "testDevice",
	}
	s.mercury = mercury.CreateMercury(&stream)
	controller := spirc.CreateController(s, []byte{})

	go controller.SendHello()

	//ignore subscribe packet
	<-stream.sendPackets

	packet := <-stream.sendPackets

	if packet.cmd != 0xb2 {
		t.Errorf("Wrong cmd code.  Got %q, want %q", packet.cmd, 0xb2)
	}

	buf := bytes.NewBuffer(packet.buf)
	checkHead(t, buf)

	frameData, _ := parsePart(buf)
	frame := &Spotify.Frame{}
	proto.Unmarshal(frameData, frame)

	if frame.GetTyp() != Spotify.MessageType_kMessageTypeHello {
		t.Errorf("Wrong message type")
	}

	if *frame.Ident != "testDevice" {
		t.Errorf("Wrong ident. Got %q, want %q", *frame.Ident, "testDevice")
	}
}
