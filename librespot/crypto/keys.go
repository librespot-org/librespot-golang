package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"log"
	"math/big"
)

type PrivateKeys struct {
	privateKey *big.Int
	publicKey  *big.Int

	generator   *big.Int
	prime       *big.Int
	clientNonce []byte
}

type SharedKeys struct {
	challenge []byte
	sendKey   []byte
	recvKey   []byte
}

func RandomVec(count int) []byte {
	c := count
	b := make([]byte, c)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal("error:", err)
	}
	return b
}

func Powm(base, exp, modulus *big.Int) *big.Int {
	exp2 := big.NewInt(0).SetBytes(exp.Bytes())
	base2 := big.NewInt(0).SetBytes(base.Bytes())
	modulus2 := big.NewInt(0).SetBytes(modulus.Bytes())
	zero := big.NewInt(0)
	result := big.NewInt(1)
	temp := new(big.Int)

	for zero.Cmp(exp2) != 0 {
		if temp.Rem(exp2, big.NewInt(2)).Cmp(zero) != 0 {
			result = result.Mul(result, base2)
			result = result.Rem(result, modulus2)
		}
		exp2 = exp2.Rsh(exp2, 1)
		base2 = base2.Mul(base2, base2)
		base2 = base2.Rem(base2, modulus2)
	}
	return result
}

func GenerateKeys() PrivateKeys {
	private := new(big.Int)
	private.SetBytes(RandomVec(95))
	nonce := RandomVec(0x10)

	return GenerateKeysFromPrivate(private, nonce)
}

func GenerateKeysFromPrivate(private *big.Int, nonce []byte) PrivateKeys {
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
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff})

	return PrivateKeys{
		privateKey: private,
		publicKey:  Powm(DH_GENERATOR, private, DH_PRIME),

		generator:   DH_GENERATOR,
		prime:       DH_PRIME,
		clientNonce: nonce,
	}
}

func (p *PrivateKeys) AddRemoteKey(remote []byte, clientPacket []byte, serverPacket []byte) SharedKeys {
	remote_be := new(big.Int)
	remote_be.SetBytes(remote)
	shared_key := Powm(remote_be, p.privateKey, p.prime)

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

	return SharedKeys{
		challenge: mac.Sum(nil),
		sendKey:   data[0x14:0x34],
		recvKey:   data[0x34:0x54],
	}
}

func (p *PrivateKeys) SharedKey(publicKey string) []byte {
	publicKeyBytes, _ := base64.StdEncoding.DecodeString(publicKey)

	publicBig := new(big.Int)
	publicBig.SetBytes(publicKeyBytes)

	sharedKey := Powm(publicBig, p.privateKey, p.prime)
	return sharedKey.Bytes()
}

func (p *PrivateKeys) PubKey() []byte {
	return p.publicKey.Bytes()
}

func (p *PrivateKeys) PrivateKey() *big.Int {
	return p.privateKey
}

func (p *PrivateKeys) Prime() *big.Int {
	return p.prime
}

func (p *PrivateKeys) ClientNonce() []byte {
	return p.clientNonce
}

func (s *SharedKeys) Challenge() []byte {
	return s.challenge
}
