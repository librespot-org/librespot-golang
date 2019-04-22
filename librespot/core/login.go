package core

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/librespot-org/librespot-golang/Spotify"
	"github.com/librespot-org/librespot-golang/librespot/connection"
	"github.com/librespot-org/librespot-golang/librespot/discovery"
	"github.com/librespot-org/librespot-golang/librespot/utils"
	"log"
)

var Version = "master"
var BuildID = "dev"

// Login to Spotify using username and password
func Login(username string, password string, deviceName string) (*Session, error) {
	s, err := setupSession()
	if err != nil {
		return s, err
	}

	return s, s.loginSession(username, password, deviceName)
}

func (s *Session) loginSession(username string, password string, deviceName string) error {
	s.deviceId = utils.GenerateDeviceId(deviceName)
	s.deviceName = deviceName

	err := s.startConnection()
	if err != nil {
		return err
	}
	loginPacket := makeLoginPasswordPacket(username, password, s.deviceId)
	return s.doLogin(loginPacket, username)
}

// Login to Spotify using an existing authData blob
func LoginSaved(username string, authData []byte, deviceName string) (*Session, error) {
	s, err := setupSession()
	if err != nil {
		return s, err
	}
	s.deviceId = utils.GenerateDeviceId(deviceName)
	s.deviceName = deviceName

	err = s.startConnection()
	if err != nil {
		return s, err
	}

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
	s, err := setupSession()
	if err != nil {
		return s, err
	}

	s.deviceId = utils.GenerateDeviceId(deviceName)
	s.deviceName = deviceName

	err = s.startConnection()
	if err != nil {
		return s, err
	}

	packet := makeLoginBlobPacket("", []byte(accessToken),
		Spotify.AuthenticationType_AUTHENTICATION_SPOTIFY_TOKEN.Enum(), s.deviceId)
	return s, s.doLogin(packet, "")
}

func (s *Session) doLogin(packet []byte, username string) error {
	err := s.stream.SendPacket(connection.PacketLogin, packet)
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
	go s.runPollLoop()

	return nil
}

func (s *Session) handleLogin() (*Spotify.APWelcome, error) {
	cmd, data, err := s.stream.RecvPacket()
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %v", err)
	}

	if cmd == connection.PacketAuthFailure {
		return nil, fmt.Errorf("authentication failed")
	} else if cmd == connection.PacketAPWelcome {
		welcome := &Spotify.APWelcome{}
		err := proto.Unmarshal(data, welcome)
		if err != nil {
			return nil, fmt.Errorf("authentication failed: %v", err)
		}
		fmt.Println("Authentication succeeded: Welcome,", welcome.GetCanonicalUsername())
		fmt.Println("Blob type:", welcome.GetReusableAuthCredentialsType())
		return welcome, nil
	} else {
		return nil, fmt.Errorf("authentication failed: unexpected cmd %v", cmd)
	}
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

func makeLoginPasswordPacket(username string, password string, deviceId string) []byte {
	return makeLoginBlobPacket(username, []byte(password),
		Spotify.AuthenticationType_AUTHENTICATION_USER_PASS.Enum(), deviceId)
}

func makeLoginBlobPacket(username string, authData []byte,
	authType *Spotify.AuthenticationType, deviceId string) []byte {
	versionString := "librespot-golang_" + Version + "_" + BuildID

	packet := &Spotify.ClientResponseEncrypted{
		LoginCredentials: &Spotify.LoginCredentials{
			Username: proto.String(username),
			Typ:      authType,
			AuthData: authData,
		},
		SystemInfo: &Spotify.SystemInfo{
			CpuFamily:               Spotify.CpuFamily_CPU_UNKNOWN.Enum(),
			Os:                      Spotify.Os_OS_UNKNOWN.Enum(),
			SystemInformationString: proto.String("librespot-golang"),
			DeviceId:                proto.String(deviceId),
		},
		VersionString: proto.String(versionString),
	}

	packetData, err := proto.Marshal(packet)
	if err != nil {
		log.Fatal("login marshaling error: ", err)
	}
	return packetData
}
