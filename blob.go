package spotcontrol

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"golang.org/x/crypto/pbkdf2"
	"log"
	"math/big"
	"os"
)

type BlobInfo struct {
	Username    string
	DecodedBlob string
}

func (b *BlobInfo) saveToFile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}

	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(b)
	if err != nil {
		return err
	}
	return nil
}

func blobFromFile(path string) (BlobInfo, error) {
	result := BlobInfo{}
	file, err := os.Open(path)
	if err != nil {
		return result, err
	}

	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&result)
	if err != nil {
		return result, err
	}
	return result, nil
}

func newBlobInfo(blob64 string, client64 string,
	keys privateKeys, deviceId string, username string) (BlobInfo, error) {

	partDecoded, err := decodeBlob(blob64, client64, keys)
	if err != nil {
		return BlobInfo{}, err
	}

	fullDecoded := decodeBlobSecondary(partDecoded, username,
		deviceId)

	return BlobInfo{
		Username:    username,
		DecodedBlob: base64.StdEncoding.EncodeToString(fullDecoded),
	}, nil
}

func (b *BlobInfo) makeAuthBlob(deviceId string, client64 string, dhKeys privateKeys) (string, error) {
	secret := sha1.Sum([]byte(deviceId))
	key := blobKey(b.Username, secret[:])

	blobBytes, err := base64.StdEncoding.DecodeString(b.DecodedBlob)
	if err != nil {
		return "", err
	}
	encoded := encryptBlob(blobBytes, key)
	fullEncoded := makeBlob(encoded, dhKeys, client64)

	return fullEncoded, nil
}

func blobKey(username string, secret []byte) []byte {
	data := pbkdf2.Key(secret, []byte(username), 0x100, 20, sha1.New)[0:20]

	hash := sha1.Sum(data)
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, 20)
	return append(hash[:], length...)
}

func makeBlob(blobPart []byte, keys privateKeys, publicKey string) string {
	part := []byte(base64.StdEncoding.EncodeToString(blobPart))

	sharedKey := keys.SharedKey(publicKey)
	iv := randomVec(16)

	key := sha1.Sum(sharedKey)
	base_key := key[:16]
	hash := hmac.New(sha1.New, base_key)

	hash.Write([]byte("checksum"))
	checksum_key := hash.Sum(nil)
	hash.Reset()

	hash.Write([]byte("encryption"))
	encryption_key := hash.Sum(nil)
	hash.Reset()

	block, _ := aes.NewCipher(encryption_key[0:16])
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(part, part)

	macHash := hmac.New(sha1.New, checksum_key)
	macHash.Write(part)
	mac := macHash.Sum(nil)

	part = append(iv, part...)
	part = append(part, mac...)

	return base64.StdEncoding.EncodeToString(part)
}

func encryptBlob(blob []byte, key []byte) []byte {
	block, _ := aes.NewCipher(key)
	bs := block.BlockSize()
	if len(blob)%bs != 0 {
		panic("Need a multiple of the blocksize")
	}

	l := len(blob)
	for i := l - 0x11; i >= 0; i-- {
		blob[l-i-1] = blob[l-i-1] ^ blob[l-i-0x11]
	}

	ciphertext := make([]byte, len(blob))
	encoded := ciphertext
	for len(blob) > 0 {
		block.Encrypt(ciphertext, blob)
		ciphertext = ciphertext[bs:]
		blob = blob[bs:]
	}

	return encoded
}

func decodeBlob(blob64 string, client64 string, keys privateKeys) (string, error) {

	clientKey, err := base64.StdEncoding.DecodeString(client64)
	if err != nil {
		return "", err
	}

	blobBytes, err := base64.StdEncoding.DecodeString(blob64)
	if err != nil {
		return "", err
	}

	clientKey_be := new(big.Int)
	clientKey_be.SetBytes(clientKey)

	sharedKey := powm(clientKey_be, keys.privateKey, keys.prime)
	iv := blobBytes[0:16]
	encryptedPart := blobBytes[16 : len(blobBytes)-20]
	ckSum := blobBytes[len(blobBytes)-20:]
	key := sha1.Sum(sharedKey.Bytes())
	base_key := key[:16]
	hash := hmac.New(sha1.New, base_key)

	hash.Write([]byte("checksum"))
	checksum_key := hash.Sum(nil)
	hash.Reset()

	hash.Write([]byte("encryption"))
	encryption_key := hash.Sum(nil)
	hash.Reset()

	macHash := hmac.New(sha1.New, checksum_key)
	macHash.Write(encryptedPart)
	mac := macHash.Sum(nil)

	if !bytes.Equal(mac, ckSum) {
		log.Println("add user error, mac doesn't match")
		return "", errors.New("mac mismatch")
	}

	block, _ := aes.NewCipher(encryption_key[0:16])
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(encryptedPart, encryptedPart)

	return string(encryptedPart), nil
}

func decodeBlobSecondary(blob64 string, username string, deviceId string) []byte {
	blob, _ := base64.StdEncoding.DecodeString(blob64)
	secret := sha1.Sum([]byte(deviceId))
	key := blobKey(username, secret[:])

	data := decryptBlob(blob, key)
	return data
}

func decryptBlob(blob []byte, key []byte) []byte {
	block, _ := aes.NewCipher(key)
	bs := block.BlockSize()
	if len(blob)%bs != 0 {
		panic("Need a multiple of the blocksize")
	}

	plaintext := make([]byte, len(blob))

	plain := plaintext
	for len(blob) > 0 {
		block.Decrypt(plaintext, blob)
		plaintext = plaintext[bs:]
		blob = blob[bs:]
	}

	l := len(plain)
	for i := 0; i < l-0x10; i++ {
		plain[l-i-1] = plain[l-i-1] ^ plain[l-i-0x11]
	}

	return plain
}
