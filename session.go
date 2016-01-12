package stringutil

import (
    "github.com/badfortrains/Spotify"
    "github.com/golang/protobuf/proto"
    "fmt"
    "net"
    "log"
    "bytes"
    "io/ioutil"
    "os"
)


type Session struct{
    stream ShannonStream
    mercury MercuryManager
}

func (s *Session) StartConnection(){
    tcpCon, err := net.Dial("tcp", "sjc1-accesspoint-a95.ap.spotify.com:4070")
    if err != nil {
        log.Fatal("Failed to coonect:", err)
    }
    conn := MakePlainConnection(tcpCon, tcpCon)

    keys := GenerateKeys()
    helloMessage := helloPacket(keys.pubKey())
    initClientPacket, err := conn.SendPrefixPacket([]byte{0,4}, helloMessage)
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
    sharedKeys := keys.addRemoteKey(remoteKey, initClientPacket, initServerPacket)

    plainResponse := &Spotify.ClientResponsePlaintext{
        LoginCryptoResponse: &Spotify.LoginCryptoResponseUnion{
            DiffieHellman: &Spotify.LoginCryptoDiffieHellmanResponse{
                Hmac: sharedKeys.challenge,
            },
        },
        PowResponse: &Spotify.PoWResponseUnion{},
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

    s.stream = SetupStream(sharedKeys, conn)
}

func (s *Session) Login(){
    username := os.Getenv("SPOT_USERNAME")
    password := os.Getenv("SPOT_PASSWORD")

    loginPacket := loginPacket("./spotify_appkey.key", username, password)
    

    err :=  s.stream.SendPacket(0xab, loginPacket)
    if err != nil {
        log.Fatal("bad shannon write", err)
    }
}

func (s *Session) Poll(mercury *MercuryManager) {
    cmd, data, err := s.stream.RecvPacket()
    if err != nil {
        log.Fatal(err)
    }
    switch {
    case cmd == 0x4:
        err = s.stream.SendPacket(0x49, data)
        if err != nil {
            log.Fatal(err)
        }
    case cmd == 0x1b:
        fmt.Println("conuntry")
    case 0xb2 < cmd && cmd < 0xb6:
        fmt.Println("mercury")
        err = mercury.handle(cmd, bytes.NewReader(data))
        if err != nil {
            log.Fatal(err)
        }
    case cmd == 0xac:
        fmt.Println("Authentication succeedded")
        
    case cmd == 0xad:
        fmt.Println("Authentication failed")
    default:
        fmt.Println("ignore")
    }
}


func loginPacket(appfile string, username string, password string) []byte{
    data, _ := ioutil.ReadFile(appfile)
    packet := &Spotify.ClientResponseEncrypted{
        LoginCredentials: &Spotify.LoginCredentials{
            Username: proto.String(username),
            Typ: Spotify.AuthenticationType_AUTHENTICATION_USER_PASS.Enum(),
            AuthData: []byte(password),
        },
        SystemInfo: &Spotify.SystemInfo{
            CpuFamily: Spotify.CpuFamily_CPU_UNKNOWN.Enum(),
            Os: Spotify.Os_OS_UNKNOWN.Enum(),
            SystemInformationString: proto.String("librespot"),
            DeviceId: proto.String("7288edd0fc3ffcbe93a0cf06e3568e28521687bc"),
        },
        VersionString: proto.String("librespot-8315e10"),
        Appkey: &Spotify.LibspotifyAppKey{
            Version: proto.Uint32(uint32(data[0])),
            Devkey: data[0x1:0x81],
            Signature: data[0x81:0x141],
            Useragent: proto.String("librespot-8315e10"),
            CallbackHash: make([]byte, 20),
        },
    }

    packetData, err := proto.Marshal(packet)
    if err != nil {
        log.Fatal("login marshaling error: ", err)
    }
    return packetData
}


func helloPacket(publicKey []byte) []byte{
    hello := &Spotify.ClientHello {
        BuildInfo: &Spotify.BuildInfo{
            Product: Spotify.Product_PRODUCT_LIBSPOTIFY_EMBEDDED.Enum(),
            Platform: Spotify.Platform_PLATFORM_LINUX_X86.Enum(),
            Version: proto.Uint64(0x10800000000),
        },
        CryptosuitesSupported: []Spotify.Cryptosuite {
            Spotify.Cryptosuite_CRYPTO_SUITE_SHANNON },
        LoginCryptoHello: &Spotify.LoginCryptoHelloUnion{
            DiffieHellman: &Spotify.LoginCryptoDiffieHellmanHello{
                Gc: publicKey,
                ServerKeysKnown: proto.Uint32(1),
            },
        },
        ClientNonce: randomVec(0x10),
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