// Package stringutil contains utility functions for working with strings.
package stringutil

// #include "./shn.h"
import "C"


import "unsafe" 
import "encoding/binary"
//void shn_key (shn_ctx * c, const UCHAR *key, int keylen);

type shnCtx C.shn_ctx

type ShannonStream struct {
    sendNonce uint32
    sendCipher C.shn_ctx

    recvNonce uint32
    recvCipher C.shn_ctx

}

func setKey(ctx *C.shn_ctx, key []uint8) {
	C.shn_key(ctx,
		(*C.uchar)(unsafe.Pointer(&key[0])), 
		C.int(len(key)))

	nonce := []uint8{0,0,0,0}
	C.shn_nonce(ctx, 
		(*C.uchar)(unsafe.Pointer(&nonce[0])), 
		C.int(len(nonce)))	
}

func (s *ShannonStream) SetRecvKey(key []uint8) {
	setKey(&s.recvCipher, key)
}

func (s *ShannonStream) SetSendKey(key []uint8) {
	setKey(&s.sendCipher, key)
}

func (s *ShannonStream) Encrypt(message string) []byte{
	messageBytes := []byte(message)
	C.shn_encrypt(&s.sendCipher,
		(*C.uchar)(unsafe.Pointer(&messageBytes[0])),
		C.int(len(messageBytes)))

	return messageBytes
}

func (s *ShannonStream) Decrypt(messageBytes []byte) []byte{
	C.shn_decrypt(&s.recvCipher,
		(*C.uchar)(unsafe.Pointer(&messageBytes[0])),
		C.int(len(messageBytes)))

	return messageBytes
}

func (s *ShannonStream) FinishSend() []byte{
	count := 4
	mac := make([]byte,count)
	C.shn_finish(&s.sendCipher,
		(*C.uchar)(unsafe.Pointer(&mac[0])),
		C.int(count))

	s.sendNonce += 1
	nonce := make([]uint8, 4)
	binary.BigEndian.PutUint32(nonce, s.sendNonce)
	C.shn_nonce(&s.sendCipher, 
		(*C.uchar)(unsafe.Pointer(&nonce[0])), 
		C.int(len(nonce)))

	return mac
}




// Reverse returns its argument string reversed rune-wise left to right.
func Reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}