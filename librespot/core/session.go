package core

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/librespot-org/librespot-golang/Spotify"
	"github.com/librespot-org/librespot-golang/librespot/connection"
	"github.com/librespot-org/librespot-golang/librespot/crypto"
	"github.com/librespot-org/librespot-golang/librespot/discovery"
	"github.com/librespot-org/librespot-golang/librespot/mercury"
	"github.com/librespot-org/librespot-golang/librespot/player"
	"github.com/librespot-org/librespot-golang/librespot/utils"
)

// Session represents an active Spotify connection
type Session struct {
	/// Constructor references
	// mercuryConstructor is the constructor that should be used to build a mercury connection
	mercuryConstructor func(conn connection.PacketStream) *mercury.Client
	// shannonConstructor is the constructor used to build the shannon-encrypted PacketStream connection
	shannonConstructor func(keys crypto.SharedKeys, conn connection.PlainConnection) connection.PacketStream

	/// Managers and helpers
	// stream is the encrypted connection to the Spotify server
	stream connection.PacketStream
	// mercury is the mercury client associated with this session
	mercury *mercury.Client
	// discovery is the discovery service used for Spotify Connect devices discovery
	discovery *discovery.Discovery
	// player is the player service used to load the audio data
	player *player.Player
	// tcpCon is the plain I/O network connection to the server
	tcpCon io.ReadWriter
	// keys are the encryption keys used to communicate with the server
	keys crypto.PrivateKeys

	/// State and variables
	// deviceId is the device identifier (computer name, Android serial number, ...) sent during auth to the Spotify
	// servers for this session
	deviceId string
	// deviceName is the device name (Android device model) sent during auth to the Spotify servers for this session
	deviceName string
	// username is the currently authenticated canonical username
	username string
	// reusableAuthBlob is the reusable authentication blob for Spotify Connect devices
	reusableAuthBlob []byte
	// country is the user country returned by the Spotify servers
	country string
}

func (s *Session) Stream() connection.PacketStream {
	return s.stream
}

func (s *Session) Discovery() *discovery.Discovery {
	return s.discovery
}

func (s *Session) Mercury() *mercury.Client {
	return s.mercury
}

func (s *Session) Player() *player.Player {
	return s.player
}

func (s *Session) Username() string {
	return s.username
}

func (s *Session) DeviceId() string {
	return s.deviceId
}

func (s *Session) ReusableAuthBlob() []byte {
	return s.reusableAuthBlob
}

func (s *Session) Country() string {
	return s.country
}

func (s *Session) startConnection() error {
	// First, start by performing a plaintext connection and send the Hello message
	conn := connection.MakePlainConnection(s.tcpCon, s.tcpCon)

	helloMessage := makeHelloMessage(s.keys.PubKey(), s.keys.ClientNonce())
	initClientPacket, err := conn.SendPrefixPacket([]byte{0, 4}, helloMessage)
	if err != nil {
		log.Fatal("Error writing client hello", err)
		return err
	}

	// Wait and read the hello reply
	initServerPacket, err := conn.RecvPacket()
	if err != nil {
		log.Fatal("Error receving packet for hello: ", err)
		return err
	}

	response := Spotify.APResponseMessage{}
	err = proto.Unmarshal(initServerPacket[4:], &response)
	if err != nil {
		log.Fatal("Failed to unmarshal server hello", err)
		return err
	}

	remoteKey := response.Challenge.LoginCryptoChallenge.DiffieHellman.Gs
	sharedKeys := s.keys.AddRemoteKey(remoteKey, initClientPacket, initServerPacket)

	plainResponse := &Spotify.ClientResponsePlaintext{
		LoginCryptoResponse: &Spotify.LoginCryptoResponseUnion{
			DiffieHellman: &Spotify.LoginCryptoDiffieHellmanResponse{
				Hmac: sharedKeys.Challenge(),
			},
		},
		PowResponse:    &Spotify.PoWResponseUnion{},
		CryptoResponse: &Spotify.CryptoResponseUnion{},
	}

	plainResponseMessage, err := proto.Marshal(plainResponse)
	if err != nil {
		log.Fatal("marshaling error: ", err)
		return err
	}

	_, err = conn.SendPrefixPacket([]byte{}, plainResponseMessage)
	if err != nil {
		log.Fatal("error writing client plain response ", err)
		return err
	}

	s.stream = s.shannonConstructor(sharedKeys, conn)
	s.mercury = s.mercuryConstructor(s.stream)

	s.player = player.CreatePlayer(s.stream, s.mercury)

	return nil
}

func setupSession() (*Session, error) {
	session := &Session{
		keys:               crypto.GenerateKeys(),
		mercuryConstructor: mercury.CreateMercury,
		shannonConstructor: crypto.CreateStream,
	}
	err := session.doConnect()

	return session, err
}

func sessionFromDiscovery(d *discovery.Discovery) (*Session, error) {
	s, err := setupSession()
	if err != nil {
		return nil, err
	}

	s.discovery = d
	s.deviceId = d.DeviceId()
	s.deviceName = d.DeviceName()

	err = s.startConnection()
	if err != nil {
		return s, err
	}

	loginPacket := s.getLoginBlobPacket(d.LoginBlob())
	return s, s.doLogin(loginPacket, d.LoginBlob().Username)
}

func (s *Session) doConnect() error {
	apUrl, err := utils.APResolve()
	if err != nil {
		log.Println("Failed to get ap url", err)
		return err
	}

	s.tcpCon, err = net.Dial("tcp", apUrl)
	if err != nil {
		log.Println("Failed to connect:", err)
		return err
	}

	return err
}

func (s *Session) disconnect() {
	if s.tcpCon != nil {
		conn := s.tcpCon.(net.Conn)
		err := conn.Close()
		if err != nil {
			log.Println("Failed to close tcp connection", err)
		}
		s.tcpCon = nil
	}
}

func (s *Session) doReconnect() error {
	s.disconnect()

	err := s.doConnect()
	if err != nil {
		return err
	}

	err = s.startConnection()
	if err != nil {
		return err
	}

	packet := makeLoginBlobPacket(s.username, s.reusableAuthBlob,
		Spotify.AuthenticationType_AUTHENTICATION_STORED_SPOTIFY_CREDENTIALS.Enum(), s.deviceId)
	return s.doLogin(packet, s.username)
}

func (s *Session) planReconnect() {
	go func() {
		time.Sleep(1 * time.Second)

		if err := s.doReconnect(); err != nil {
			// Try to reconnect again in a second
			s.planReconnect()
		}
	}()
}

func (s *Session) runPollLoop() {
	for {
		cmd, data, err := s.stream.RecvPacket()
		if err != nil {
			log.Println("Error during RecvPacket: ", err)

			if err == io.EOF {
				// We've been disconnected, reconnect
				s.planReconnect()
				break
			}
		} else {
			s.handle(cmd, data)
		}
	}
}

func (s *Session) handle(cmd uint8, data []byte) {
	//fmt.Printf("handle, cmd=0x%x data=%x\n", cmd, data)

	switch {
	case cmd == connection.PacketPing:
		// Ping
		err := s.stream.SendPacket(connection.PacketPong, data)
		if err != nil {
			log.Fatal("Error handling PacketPing", err)
		}

	case cmd == connection.PacketPongAck:
		// Pong reply, ignore

	case cmd == connection.PacketAesKey || cmd == connection.PacketAesKeyError ||
		cmd == connection.PacketStreamChunkRes:
		// Audio key and data responses
		s.player.HandleCmd(cmd, data)

	case cmd == connection.PacketCountryCode:
		// Handle country code
		s.country = fmt.Sprintf("%s", data)

	case 0xb2 <= cmd && cmd <= 0xb6:
		// Mercury responses
		err := s.mercury.Handle(cmd, bytes.NewReader(data))
		if err != nil {
			log.Fatal("Handle 0xbx", err)
		}

	case cmd == connection.PacketSecretBlock:
		// Old RSA public key

	case cmd == connection.PacketLegacyWelcome:
		// Empty welcome packet

	case cmd == connection.PacketProductInfo:
		// Has some info about A/B testing status, product setup, etc... in an XML fashion.

	case cmd == 0x1f:
		// Unknown, data is zeroes only

	case cmd == connection.PacketLicenseVersion:
		// This is a simple blob containing the current Spotify license version (e.g. 1.0.1-FR). Format of the blob
		// is [ uint16 id (= 0x001), uint8 len, string license ]

	default:
		fmt.Printf("Unhandled cmd 0x%x\n", cmd)
	}
}

func (s *Session) poll() {
	cmd, data, err := s.stream.RecvPacket()
	if err != nil {
		log.Fatal("poll error", err)
	}
	s.handle(cmd, data)
}

func readInt(b *bytes.Buffer) uint32 {
	c, _ := b.ReadByte()
	lo := uint32(c)
	if lo&0x80 == 0 {
		return lo
	}

	c2, _ := b.ReadByte()
	hi := uint32(c2)
	return lo&0x7f | hi<<7
}

func readBytes(b *bytes.Buffer) []byte {
	length := readInt(b)
	data := make([]byte, length)
	b.Read(data)

	return data
}

func makeHelloMessage(publicKey []byte, nonce []byte) []byte {
	hello := &Spotify.ClientHello{
		BuildInfo: &Spotify.BuildInfo{
			Product:  Spotify.Product_PRODUCT_PARTNER.Enum(),
			Platform: Spotify.Platform_PLATFORM_LINUX_X86.Enum(),
			Version:  proto.Uint64(0x10800000000),
		},
		CryptosuitesSupported: []Spotify.Cryptosuite{
			Spotify.Cryptosuite_CRYPTO_SUITE_SHANNON},
		LoginCryptoHello: &Spotify.LoginCryptoHelloUnion{
			DiffieHellman: &Spotify.LoginCryptoDiffieHellmanHello{
				Gc:              publicKey,
				ServerKeysKnown: proto.Uint32(1),
			},
		},
		ClientNonce: nonce,
		FeatureSet: &Spotify.FeatureSet{
			Autoupdate2: proto.Bool(true),
		},
		Padding: []byte{0x1e},
	}

	packetData, err := proto.Marshal(hello)
	if err != nil {
		log.Fatal("login marshaling error: ", err)
	}

	return packetData
}
