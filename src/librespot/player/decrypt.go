package player

import (
	"crypto/aes"
	"crypto/cipher"
)

var AUDIO_AESIV = []byte{0x72, 0xe0, 0x67, 0xfb, 0xdd, 0xcb, 0xcf, 0x77, 0xeb, 0xe8, 0xbc, 0x64, 0x3f, 0x63, 0x0d, 0x93}

func createCipher() cipher.Block {
	key := []byte(AUDIO_AESIV)
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	return block
}

func decryptAudio(ciphertext []byte) []byte {
	block := createCipher()
	iv := ciphertext[:aes.BlockSize]

	plaintext := make([]byte, len(ciphertext)-aes.BlockSize)
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(plaintext, ciphertext[aes.BlockSize:])

	return plaintext
}
