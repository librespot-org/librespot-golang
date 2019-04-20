package utils

import (
	"crypto/sha1"
	"encoding/base64"
)

func GenerateDeviceId(name string) string {
	hash := sha1.Sum([]byte(name))
	hash64 := base64.StdEncoding.EncodeToString(hash[:])
	return hash64
}
