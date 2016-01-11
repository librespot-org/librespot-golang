        // let request = protobuf_init!(protocol::keyexchange::ClientHello::new(), {
        //     build_info => {
        //         product: protocol::keyexchange::Product::PRODUCT_LIBSPOTIFY_EMBEDDED,
        //         platform: protocol::keyexchange::Platform::PLATFORM_LINUX_X86,
        //         version: 0x10800000000,
        //     },
        //     /*
        //     fingerprints_supported => [
        //         protocol::keyexchange::Fingerprint::FINGERPRINT_GRAIN
        //     ],
        //     */
        //     cryptosuites_supported => [
        //         protocol::keyexchange::Cryptosuite::CRYPTO_SUITE_SHANNON,
        //         //protocol::keyexchange::Cryptosuite::CRYPTO_SUITE_RC4_SHA1_HMAC
        //     ],
        //     /*
        //     powschemes_supported => [
        //         protocol::keyexchange::Powscheme::POW_HASH_CASH
        //     ],
        //     */
        //     login_crypto_hello.diffie_hellman => {
        //         gc: keys.public_key(),
        //         server_keys_known: 1,
        //     },
        //     client_nonce: util::rand_vec(&mut thread_rng(), 0x10),
        //     padding: vec![0x1e],
        //     feature_set => {
        //         autoupdate2: true,
        //     }
        // });
package stringutil

import (
    "github.com/badfortrains/Spotify"
    "github.com/golang/protobuf/proto"
    "fmt"
    "encoding/binary"
    "net"
    "io"
    "log"
)


    // pub fn send_packet_prefix(&mut self, prefix: &[u8], data: &[u8]) -> Result<Vec<u8>> {
    //     let size = prefix.len() + 4 + data.len();
    //     let mut buf = Vec::with_capacity(size);

    //     try!(buf.write(prefix));
    //     try!(buf.write_u32::<BigEndian>(size as u32));
    //     try!(buf.write(data));
    //     try!(self.stream.write(&buf));
    //     try!(self.stream.flush());

    //     Ok(buf)
    // }

func makePacketPrefix(prefix []byte, data []byte) []byte{
     fmt.Println("yo hii")
    size := len(prefix) + 4 + len(data)
    buf := make([]byte, 0, size)
    buf = append(buf, prefix...)
    sizeBuf := make([]byte, 4)
    binary.BigEndian.PutUint32(sizeBuf, uint32(size))
    buf = append(buf, sizeBuf...)
    fmt.Println("hi")
    return append(buf, data...)
}

    // pub fn recv_packet(&mut self) -> Result<Vec<u8>> {
    //     let size = try!(self.stream.read_u32::<BigEndian>()) as usize;
    //     let mut buffer = vec![0u8; size];

    //     BigEndian::write_u32(&mut buffer, size as u32);
    //     try!(self.stream.read_exact(&mut buffer[4..]));

    //     Ok(buffer)
    // }


        // let size = try!(self.stream.read_u32::<BigEndian>()) as usize;
        // let mut buffer = vec![0u8; size];

        // BigEndian::write_u32(&mut buffer, size as u32);
        // try!(self.stream.read_exact(&mut buffer[4..]));

        // Ok(buffer)

func recvPacket(conn io.Reader) []byte {
    var size uint32;
    err := binary.Read(conn, binary.BigEndian, &size);
    if err != nil {
        log.Fatal("bad response packet", err)
    }
    fmt.Println("got response, need to read %v", size)
    buf := make([]byte, size - 4)
    _, err = io.ReadFull(conn, buf)
    if err != nil {
        log.Fatal("Wrong number of bytes in response", err)
    }
    return buf
}


func connect() {
    conn, err := net.Dial("tcp", "lon3-accesspoint-a26.ap.spotify.com:4070")
    if err != nil {
        log.Fatal("Failed to coonect:", err)
    }

    keys := GenerateKeys()

    data, err := proto.Marshal(CreateHello(keys.pubKey()))
    if err != nil {
        log.Fatal("marshaling error: ", err)
    }

    initClientPacket := makePacketPrefix([]byte{0,4},data)
    fmt.Println("%v", initClientPacket)
    _, err = conn.Write(initClientPacket)
    if err != nil {
        log.Fatal("error writing client hello ", err)
    }
    fmt.Println("wrote the data")

    initServerPacket := recvPacket(conn)
    fmt.Println("read it all")
    response := &Spotify.APResponseMessage{}
    err = proto.Unmarshal(initServerPacket, response)
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


    data, err = proto.Marshal(plainResponse)
    if err != nil {
        log.Fatal("marshaling error: ", err)
    }
    fmt.Println("send res length: ", len(data), data)
    _, err = conn.Write(data)
    if err != nil {
        log.Fatal("error writing client plain response ", err)
    }

    shannon := ShannonStream{}
    shannon.SetSendKey(sharedKeys.sendKey)
    shannon.SetRecvKey(sharedKeys.recvKey)
    shannon.WrapReader(conn)

    recvPacket(&shannon)

}

func CreateHello(publicKey []byte) *Spotify.ClientHello{

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
                //Gc: []byte{144, 34, 28, 37, 124, 62, 234, 83, 128, 164, 233, 207, 228, 135, 238, 212, 241, 93, 47, 113, 103, 209, 58, 10, 152, 190, 244, 48, 3, 81, 65, 80, 20, 25, 251, 224, 225, 133, 37, 84, 135, 210, 6, 241, 183, 237, 11, 242, 82, 233, 199, 8, 207, 98, 206, 1, 33, 237, 131, 252, 198, 131, 139, 181, 137, 13, 125, 169, 173, 156, 236, 217, 13, 242, 198, 195, 66, 19, 229, 121, 228, 153, 14, 243, 24, 86, 191, 46, 241, 41, 172, 144, 130, 218, 127, 148},
                //Gc: keys.pubKey(),
                ServerKeysKnown: proto.Uint32(1),
            },
        },
        ClientNonce: randomVec(0x10),
        //Padding: make([]byte, 0x1e),
        FeatureSet: &Spotify.FeatureSet{
            Autoupdate2: proto.Bool(true),
        },
    }


    return hello
}