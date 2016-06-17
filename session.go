// Package spotcontol contains functions to remotely
// control spotify connect devices.
package spotcontrol

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	Spotify "github.com/badfortrains/spotcontrol/proto"
	"github.com/golang/protobuf/proto"
	"io"
	"io/ioutil"
	"log"
	"net"
)

const (
	subscribe_type = iota
	request_type
)

type mercuryCon interface {
	Subscribe(uri string, recv chan mercuryResponse) error
	request(req mercuryRequest, cb responseCallback) (err error)
	handle(cmd uint8, reader io.Reader) (err error)
}

type packetStream interface {
	SendPacket(cmd uint8, data []byte) (err error)
	RecvPacket() (cmd uint8, buf []byte, err error)
}

type dialer interface {
	Dial(network, address string) (net.Conn, error)
}

//Represents an active authenticated spotify connection
type session struct {
	stream  packetStream
	mercury mercuryCon
	tcpCon  io.ReadWriter
	keys    privateKeys

	mercuryConstructor func(s *session) mercuryCon
	shannonConstructor func(keys sharedKeys, conn plainConnection) packetStream

	discovery *discovery

	deviceId   string
	deviceName string
}

func (s *session) startConnection() {
	conn := makePlainConnection(s.tcpCon, s.tcpCon)
	helloMessage := helloPacket(s.keys.pubKey(), s.keys.clientNonce)
	initClientPacket, err := conn.SendPrefixPacket([]byte{0, 4}, helloMessage)
	if err != nil {
		log.Fatal("error writing client hello ", err)
	}

	initServerPacket, _ := conn.RecvPacket()
	response := &Spotify.APResponseMessage{}
	err = proto.Unmarshal(initServerPacket[4:], response)
	if err != nil {
		log.Fatal("failed to Unmarshal server packet")
	}

	remoteKey := response.Challenge.LoginCryptoChallenge.DiffieHellman.Gs
	sharedKeys := s.keys.addRemoteKey(remoteKey, initClientPacket, initServerPacket)

	plainResponse := &Spotify.ClientResponsePlaintext{
		LoginCryptoResponse: &Spotify.LoginCryptoResponseUnion{
			DiffieHellman: &Spotify.LoginCryptoDiffieHellmanResponse{
				Hmac: sharedKeys.challenge,
			},
		},
		PowResponse:    &Spotify.PoWResponseUnion{},
		CryptoResponse: &Spotify.CryptoResponseUnion{},
	}

	plainResponsMessage, err := proto.Marshal(plainResponse)
	if err != nil {
		log.Fatal("marshaling error: ", err)
	}

	_, err = conn.SendPrefixPacket([]byte{}, plainResponsMessage)
	if err != nil {
		log.Fatal("error writing client plain response ", err)
	}

	s.stream = s.shannonConstructor(sharedKeys, conn)
	s.mercury = s.mercuryConstructor(s)
}

func (s *session) doLogin(packet []byte, username string) *SpircController {
	err := s.stream.SendPacket(0xab, packet)
	if err != nil {
		log.Fatal("bad shannon write", err)
	}

	//poll once for authentication response
	s.poll()
	go s.run()

	return setupController(s, username)
}

func generateDeviceId(name string) string {
	hash := sha1.Sum([]byte(name))
	hash64 := base64.StdEncoding.EncodeToString(hash[:])
	return hash64
}

//Login to spotify with supplied byte slice for app key
func LoginWithKey(username string, password string, appkey []byte, deviceName string) *SpircController {
	s := setupSession()

	return s.loginSession(username, password, appkey, deviceName)
}

//Login to spotify using username, password and app key file.
func Login(username string, password string, appkeyPath string, deviceName string) *SpircController {
	data, err := ioutil.ReadFile(appkeyPath)
	if err != nil {
		log.Fatal("failed to open spotify appkey file")
	}
	return LoginWithKey(username, password, data, deviceName)
}

func (s *session) loginSession(username string, password string,
	appkey []byte, deviceName string) *SpircController {
	s.deviceId = generateDeviceId(deviceName)
	s.deviceName = deviceName

	s.startConnection()
	loginPacket := loginPacketPassword(appkey, username, password, s.deviceId)
	return s.doLogin(loginPacket, username)
}

func LoginBlob(username string, blob string, appkey []byte, deviceName string) *SpircController {
	deviceId := generateDeviceId(deviceName)
	discovery := discoveryFromBlob(BlobInfo{
		Username:    username,
		DecodedBlob: blob,
	}, "", deviceId, deviceName)
	return sessionFromDiscovery(discovery, appkey)
}

func setupSession() *session {
	apUrl, err := apresolve()
	if err != nil {
		log.Fatal("Failed to get ap url", err)
	}

	tcpCon, err := net.Dial("tcp", apUrl)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	return &session{
		keys:               generateKeys(),
		tcpCon:             tcpCon,
		mercuryConstructor: setupMercury,
		shannonConstructor: setupStream,
	}
}

func sessionFromDiscovery(d *discovery, appkey []byte) *SpircController {
	s := setupSession()
	s.discovery = d
	s.deviceId = d.deviceId
	s.deviceName = s.deviceName

	s.startConnection()
	loginPacket := s.getLoginBlobPacket(appkey, d.loginBlob)
	return s.doLogin(loginPacket, d.loginBlob.Username)
}

//Registers spotcontrol as a spotify conenct device via mdns.
//When user connects, logs on to spotify and saves credentials
//in file at cacheBlobPath.
//Once saved, the blob credentials allow the program
//to connect to other spotify connect devices and control them.
func LoginDiscovery(cacheBlobPath, appkeyPath string, deviceName string) *SpircController {
	deviceId := generateDeviceId(deviceName)
	discovery := loginFromConnect(cacheBlobPath, deviceId, deviceName)
	appkey, err := ioutil.ReadFile(appkeyPath)
	if err != nil {
		log.Fatal("failed to open spotify appkey file")
	}
	return sessionFromDiscovery(discovery, appkey)
}

//Login from credentials at cacheBlobPath previously saved
//by LoginDiscovery.
func LoginBlobFile(cacheBlobPath, appkeyPath string, deviceName string) *SpircController {
	deviceId := generateDeviceId(deviceName)
	discovery := loginFromFile(cacheBlobPath, deviceId, deviceName)
	appkey, err := ioutil.ReadFile(appkeyPath)
	if err != nil {
		log.Fatal("failed to open spotify appkey file")
	}
	return sessionFromDiscovery(discovery, appkey)
}

type cmdPkt struct {
	cmd  uint8
	data []byte
}

func (s *session) run() {
	for {
		cmd, data, err := s.stream.RecvPacket()
		if err != nil {
			log.Fatal("run error", err)
		}
		s.handle(cmd, data)
	}
}

func (s *session) mercurySubscribe(uri string, responseCh chan mercuryResponse) error {
	return s.mercury.Subscribe(uri, responseCh)
}

func (s *session) mercurySendRequest(request mercuryRequest, responseCb responseCallback) error {
	return s.mercury.request(request, responseCb)
}

func (s *session) handle(cmd uint8, data []byte) {
	switch {
	case cmd == 0x4:
		err := s.stream.SendPacket(0x49, data)
		if err != nil {
			log.Fatal("handle 0x4", err)
		}
	case cmd == 0x1b:
	case 0xb2 <= cmd && cmd <= 0xb6:
		err := s.mercury.handle(cmd, bytes.NewReader(data))
		if err != nil {
			log.Fatal("handle 0xbx", err)
		}
	case cmd == 0xac:
		welcome := &Spotify.APWelcome{}
		err := proto.Unmarshal(data, welcome)
		if err != nil {
			log.Fatal("handle 0xac", err)
		}
		fmt.Println("Authentication succeedded: ", welcome.GetCanonicalUsername())

	case cmd == 0xad:
		fmt.Println("Authentication failed")
	default:
	}
}

func (s *session) poll() {
	cmd, data, err := s.stream.RecvPacket()
	if err != nil {
		log.Fatal("poll error", err)
	}
	s.handle(cmd, data)
}

func (s *session) getLoginBlobPacket(appkey []byte, blob BlobInfo) []byte {
	data, _ := base64.StdEncoding.DecodeString(blob.DecodedBlob)

	buffer := bytes.NewBuffer(data)
	buffer.ReadByte()
	readBytes(buffer)
	buffer.ReadByte()
	authNum := readInt(buffer)
	authType := Spotify.AuthenticationType(authNum)
	buffer.ReadByte()
	authData := readBytes(buffer)

	return loginPacket(appkey, blob.Username, authData, &authType, s.deviceId)
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

func loginPacketPassword(appkey []byte, username, password, deviceId string) []byte {
	return loginPacket(appkey, username, []byte(password),
		Spotify.AuthenticationType_AUTHENTICATION_USER_PASS.Enum(), deviceId)
}

func loginPacket(appkey []byte, username string, authData []byte,
	authType *Spotify.AuthenticationType, deviceId string) []byte {

	fmt.Println("do login", deviceId)

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
		Appkey: &Spotify.LibspotifyAppKey{
			Version:      proto.Uint32(uint32(appkey[0])),
			Devkey:       appkey[0x1:0x81],
			Signature:    appkey[0x81:0x141],
			Useragent:    proto.String("librespot-8315e10"),
			CallbackHash: make([]byte, 20),
		},
	}

	packetData, err := proto.Marshal(packet)
	if err != nil {
		log.Fatal("login marshaling error: ", err)
	}
	return packetData
}

func helloPacket(publicKey []byte, nonce []byte) []byte {
	hello := &Spotify.ClientHello{
		BuildInfo: &Spotify.BuildInfo{
			Product:  Spotify.Product_PRODUCT_LIBSPOTIFY_EMBEDDED.Enum(),
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
	}

	packetData, err := proto.Marshal(hello)
	if err != nil {
		log.Fatal("login marshaling error: ", err)
	}

	return packetData
}
