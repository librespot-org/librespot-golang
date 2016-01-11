package stringutil

import (
    "math/big"
    "crypto/rand"
    "fmt"
    "crypto/hmac"
    "crypto/sha1"
)

type PrivateKeys struct {
    privateKey *big.Int
    publicKey *big.Int

    generator *big.Int
    prime *big.Int
}

type SharedKeys struct {
    challenge []byte
    sendKey []byte
    recvKey []byte
}

func randomVec(count int) []byte{
    c := count
    b := make([]byte, c)
    _, err := rand.Read(b)
    if err != nil {
        fmt.Println("error:", err)
    }
    return b 
}
//powm(base: &BigUint, exp: &BigUint, modulus: &BigUint) 
// let mut base = base.clone();
// let mut exp = exp.clone();
// let mut result : BigUint = One::one();

// while !exp.is_zero() {
//     if exp.is_odd() {
//         result = result.mul(&base).rem(modulus);
//     }
//     exp = exp.shr(1);
//     base = (&base).mul(&base).rem(modulus);
// }

// return result;

func powm(base, exp, modulus *big.Int) *big.Int{
    zero := big.NewInt(0)
    result := big.NewInt(1)
    temp := new(big.Int)
    for exp.Cmp(zero) != 0 {
        if temp.Rem(exp, big.NewInt(2)).Cmp(zero) != 0 {
            result = result.Mul(result, base)
            result = result.Rem(result, modulus)
        }
        exp = exp.Rsh(exp, 1)
        base = base.Mul(base, base)
        base = base.Rem(base, modulus)
    }
    return result
}

func GenerateKeys() PrivateKeys{
    DH_GENERATOR := big.NewInt(0x2)
    DH_PRIME := new(big.Int)
    DH_PRIME.SetBytes([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xc9,
            0x0f, 0xda, 0xa2, 0x21, 0x68, 0xc2, 0x34, 0xc4, 0xc6,
            0x62, 0x8b, 0x80, 0xdc, 0x1c, 0xd1, 0x29, 0x02, 0x4e,
            0x08, 0x8a, 0x67, 0xcc, 0x74, 0x02, 0x0b, 0xbe, 0xa6,
            0x3b, 0x13, 0x9b, 0x22, 0x51, 0x4a, 0x08, 0x79, 0x8e,
            0x34, 0x04, 0xdd, 0xef, 0x95, 0x19, 0xb3, 0xcd, 0x3a,
            0x43, 0x1b, 0x30, 0x2b, 0x0a, 0x6d, 0xf2, 0x5f, 0x14,
            0x37, 0x4f, 0xe1, 0x35, 0x6d, 0x6d, 0x51, 0xc2, 0x45,
            0xe4, 0x85, 0xb5, 0x76, 0x62, 0x5e, 0x7e, 0xc6, 0xf4,
            0x4c, 0x42, 0xe9, 0xa6, 0x3a, 0x36, 0x20, 0xff, 0xff,
            0xff, 0xff, 0xff, 0xff, 0xff, 0xff });
    private := new(big.Int)
    private.SetBytes(randomVec(95))


    return PrivateKeys{
        privateKey: private,
        publicKey: powm(DH_GENERATOR, private, DH_PRIME),

        generator: DH_GENERATOR,
        prime: DH_PRIME,
    }
}

    // pub fn add_remote_key(self, remote_key: &[u8], client_packet: &[u8], server_packet: &[u8]) -> SharedKeys {
    //     let shared_key = util::powm(&BigUint::from_bytes_be(remote_key), &self.private_key, &DH_PRIME);

    //     let mut data = Vec::with_capacity(0x64);
    //     let mut mac = crypto::hmac::Hmac::new(crypto::sha1::Sha1::new(), &shared_key.to_bytes_be());

    //     for i in 1..6 {
    //         mac.input(client_packet);
    //         mac.input(server_packet);
    //         mac.input(&[i]);
    //         data.write(&mac.result().code()).unwrap();
    //         mac.reset();
    //     }

    //     mac = crypto::hmac::Hmac::new(crypto::sha1::Sha1::new(), &data[..0x14]);
    //     mac.input(client_packet);
    //     mac.input(server_packet);

    //     SharedKeys {
    //         //private: self,
    //         challenge: mac.result().code().to_vec(),
    //         send_key: data[0x14..0x34].to_vec(),
    //         recv_key: data[0x34..0x54].to_vec(),
    //     }
    // }
func (p *PrivateKeys) addRemoteKey(remote []byte, clientPacket []byte, serverPacket []byte) SharedKeys{
    remote_be := new(big.Int)
    remote_be.SetBytes(remote)
    shared_key := powm(remote_be, p.privateKey, p.prime)

    data := make([]byte, 0, 100)
    mac := hmac.New(sha1.New, shared_key.Bytes())

    for i := 1; i < 6; i++ {
        mac.Write(clientPacket)
        mac.Write(serverPacket)
        mac.Write([]byte{uint8(i)})
        data = append(data, mac.Sum(nil)...)
        mac.Reset()
    }

    mac = hmac.New(sha1.New, data[0:0x14])
    mac.Write(clientPacket)
    mac.Write(serverPacket)

    fmt.Println("data length", len(data))

    return SharedKeys{
        challenge: mac.Sum(nil),
        sendKey: data[0x14:0x34],
        recvKey: data[0x34:0x54],
    }

}

func (p *PrivateKeys) pubKey() []byte{
    return p.publicKey.Bytes()
}