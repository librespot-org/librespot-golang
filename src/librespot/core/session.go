package core

import (
	"Spotify"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"github.com/golang/protobuf/proto"
	"io"
	"librespot/connection"
	"librespot/crypto"
	"librespot/discovery"
	"librespot/mercury"
	"librespot/player"
	"librespot/utils"
	"log"
	"net"
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

// Login to Spotify using username and password
func Login(username string, password string, deviceName string) (*Session, error) {
	s := setupSession()
	return s, s.loginSession(username, password, deviceName)
}

func (s *Session) loginSession(username string, password string, deviceName string) error {
	s.deviceId = utils.GenerateDeviceId(deviceName)
	s.deviceName = deviceName

	s.startConnection()
	loginPacket := makeLoginPasswordPacket(username, password, s.deviceId)
	return s.doLogin(loginPacket, username)
}

// Login to Spotify using an existing authData blob
func LoginSaved(username string, authData []byte, deviceName string) (*Session, error) {
	s := setupSession()
	s.deviceId = utils.GenerateDeviceId(deviceName)
	s.deviceName = deviceName

	s.startConnection()
	packet := makeLoginBlobPacket(username, authData,
		Spotify.AuthenticationType_AUTHENTICATION_STORED_SPOTIFY_CREDENTIALS.Enum(), s.deviceId)
	return s, s.doLogin(packet, username)
}

// Registers librespot as a Spotify Connect device via mdns. When user connects, logs on to Spotify and saves
// credentials in file at cacheBlobPath. Once saved, the blob credentials allow the program to connect to other
// Spotify Connect devices and control them.
func LoginDiscovery(cacheBlobPath string, deviceName string) (*Session, error) {
	deviceId := utils.GenerateDeviceId(deviceName)
	disc := discovery.LoginFromConnect(cacheBlobPath, deviceId, deviceName)
	return sessionFromDiscovery(disc)
}

// Login using an authentication blob through Spotify Connect discovery system, reading an existing blob data. To read
// from a file, see LoginDiscoveryBlobFile.
func LoginDiscoveryBlob(username string, blob string, deviceName string) (*Session, error) {
	deviceId := utils.GenerateDeviceId(deviceName)
	disc := discovery.CreateFromBlob(utils.BlobInfo{
		Username:    username,
		DecodedBlob: blob,
	}, "", deviceId, deviceName)
	return sessionFromDiscovery(disc)
}

// Login from credentials at cacheBlobPath previously saved by LoginDiscovery. Similar to LoginDiscoveryBlob, except
// it reads it directly from a file.
func LoginDiscoveryBlobFile(cacheBlobPath, deviceName string) (*Session, error) {
	deviceId := utils.GenerateDeviceId(deviceName)
	disc := discovery.CreateFromFile(cacheBlobPath, deviceId, deviceName)
	return sessionFromDiscovery(disc)
}

// Login to Spotify using the OAuth method
func LoginOAuth(deviceName string, clientId string, clientSecret string) (*Session, error) {
	token := getOAuthToken(clientId, clientSecret)
	return loginOAuthToken(token.AccessToken, deviceName)
}

func loginOAuthToken(accessToken string, deviceName string) (*Session, error) {
	s := setupSession()
	s.deviceId = utils.GenerateDeviceId(deviceName)
	s.deviceName = deviceName

	s.startConnection()

	packet := makeLoginBlobPacket("", []byte(accessToken),
		Spotify.AuthenticationType_AUTHENTICATION_SPOTIFY_TOKEN.Enum(), s.deviceId)
	return s, s.doLogin(packet, "")
}

func (s *Session) doLogin(packet []byte, username string) error {
	err := s.stream.SendPacket(0xab, packet)
	if err != nil {
		log.Fatal("bad shannon write", err)
	}

	// Pll once for authentication response
	welcome, err := s.handleLogin()
	if err != nil {
		return err
	}

	// Store the few interesting values
	s.username = welcome.GetCanonicalUsername()
	if s.username == "" {
		// Spotify might not return a canonical username, so reuse the blob's one instead
		s.username = s.discovery.LoginBlob().Username
	}
	s.reusableAuthBlob = welcome.GetReusableAuthCredentials()

	// Poll for acknowledge before loading - needed for gopherjs
	// s.poll()
	go s.run()

	return nil
}

func (s *Session) getAudioFile(fileId []byte, trackId []byte, start uint32, end uint32) {
	// Request the audio key (cipher)
	buf := new(bytes.Buffer)

	buf.Write(fileId)
	buf.Write(trackId)
	buf.Write(s.mercury.NextSeq())
	binary.Write(buf, binary.BigEndian, uint16(0x0000))

	err := s.stream.SendPacket(0xc, buf.Bytes())

	if err != nil {
		log.Println("Error while sending packet", err)
	}
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
		log.Fatal("Error receving packet for hello", err)
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

	plainResponsMessage, err := proto.Marshal(plainResponse)
	if err != nil {
		log.Fatal("marshaling error: ", err)
		return err
	}

	_, err = conn.SendPrefixPacket([]byte{}, plainResponsMessage)
	if err != nil {
		log.Fatal("error writing client plain response ", err)
		return err
	}

	s.stream = s.shannonConstructor(sharedKeys, conn)
	s.mercury = s.mercuryConstructor(s.stream)

	s.player = player.CreatePlayer(s.stream, s.mercury)

	return nil
}

func setupSession() *Session {
	apUrl, err := utils.APResolve()
	if err != nil {
		log.Fatal("Failed to get ap url", err)
	}

	tcpCon, err := net.Dial("tcp", apUrl)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	return &Session{
		keys:               crypto.GenerateKeys(),
		tcpCon:             tcpCon,
		mercuryConstructor: mercury.CreateMercury,
		shannonConstructor: crypto.CreateStream,
	}
}

func sessionFromDiscovery(d *discovery.Discovery) (*Session, error) {
	s := setupSession()
	s.discovery = d
	s.deviceId = d.DeviceId()
	s.deviceName = d.DeviceName()

	s.startConnection()
	loginPacket := s.getLoginBlobPacket(d.LoginBlob())
	return s, s.doLogin(loginPacket, d.LoginBlob().Username)
}

func (s *Session) run() {
	for {
		cmd, data, err := s.stream.RecvPacket()
		if err != nil {
			log.Fatal("Error during RecvPacket: ", err)
		}

		s.handle(cmd, data)
	}
}

/*
func (s *Session) mercurySubscribe(uri string, responseCh chan mercury.Response, responseCb mercury.Callback) error {
	return s.mercury.Subscribe(uri, responseCh, responseCb)
}

func (s *Session) mercurySendRequest(request mercury.Request, responseCb mercury.Callback) {
	err := s.mercury.Request(request, responseCb)
	if err != nil && responseCb != nil {
		responseCb(mercury.Response{
			StatusCode: 500,
		})
	}
}
*/
func (s *Session) handleLogin() (*Spotify.APWelcome, error) {
	cmd, data, err := s.stream.RecvPacket()
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %v", err)
	}

	if cmd == 0xad {
		return nil, fmt.Errorf("authentication failed")
	} else if cmd == 0xac {
		welcome := &Spotify.APWelcome{}
		err := proto.Unmarshal(data, welcome)
		if err != nil {
			return nil, fmt.Errorf("authentication failed: %v", err)
		}
		fmt.Println("authentication succeedded: ", welcome.GetCanonicalUsername())
		fmt.Println("got type", welcome.GetReusableAuthCredentialsType())
		return welcome, nil
	} else {
		return nil, fmt.Errorf("authentication failed: unexpected cmd %v", cmd)
	}
}

func (s *Session) handle(cmd uint8, data []byte) {
	//fmt.Printf("handle, cmd=0x%x data len=%d\n", cmd, len(data))

	switch {
	case cmd == kPacketPing:
		// Ping
		err := s.stream.SendPacket(kPacketPong, data)
		if err != nil {
			log.Fatal("Error handling kPacketPing", err)
		}

	case cmd == kPacketPongAck:
		// Pong reply, ignore

	case cmd == kPacketAesKey || cmd == kPacketAesKeyError ||
		cmd == kPacketStreamChunk:
		// Audio key and data responses
		s.player.HandleCmd(cmd, data)

	case cmd == kPacketCountryCode:
		// Handle country code
		s.country = fmt.Sprintf("%s", data)

	case 0xb2 <= cmd && cmd <= 0xb6:
		// Mercury responses
		err := s.mercury.Handle(cmd, bytes.NewReader(data))
		if err != nil {
			log.Fatal("Handle 0xbx", err)
		}

	case cmd == kPacketSecretBlock:
		// Old RSA public key

	case cmd == kPacketLegacyWelcome:
		// Empty welcome packet

	case cmd == kPacketProductInfo:
		// Has some info about A/B testing status, product setup, etc... in an XML fashion.

	case cmd == 0x1f:
		// Unknown, data is zeroes only

	case cmd == kPacketLicenseVersion:
		// This is a simple blob containing the current spotify license. Format of the blob
		// is [ uint16 id, uint8 len, string license ]

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

func (s *Session) getLoginBlobPacket(blob utils.BlobInfo) []byte {
	data, _ := base64.StdEncoding.DecodeString(blob.DecodedBlob)

	buffer := bytes.NewBuffer(data)
	buffer.ReadByte()
	readBytes(buffer)
	buffer.ReadByte()
	authNum := readInt(buffer)
	authType := Spotify.AuthenticationType(authNum)
	buffer.ReadByte()
	authData := readBytes(buffer)

	return makeLoginBlobPacket(blob.Username, authData, &authType, s.deviceId)
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

func makeLoginPasswordPacket(username string, password string, deviceId string) []byte {
	return makeLoginBlobPacket(username, []byte(password),
		Spotify.AuthenticationType_AUTHENTICATION_USER_PASS.Enum(), deviceId)
}

func makeLoginBlobPacket(username string, authData []byte,
	authType *Spotify.AuthenticationType, deviceId string) []byte {

	packet := &Spotify.ClientResponseEncrypted{
		LoginCredentials: &Spotify.LoginCredentials{
			Username: proto.String(username),
			Typ:      authType,
			AuthData: authData,
		},
		SystemInfo: &Spotify.SystemInfo{
			CpuFamily: Spotify.CpuFamily_CPU_UNKNOWN.Enum(),
			Os:        Spotify.Os_OS_UNKNOWN.Enum(),
			SystemInformationString: proto.String("librespot"),
			DeviceId:                proto.String(deviceId),
		},
		VersionString: proto.String("librespot-8315e10"),
	}

	packetData, err := proto.Marshal(packet)
	if err != nil {
		log.Fatal("login marshaling error: ", err)
	}
	return packetData
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
