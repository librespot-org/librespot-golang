package librespot

import (
	core "github.com/librespot-org/librespot-golang/librespot/core"
)

// Login to Spotify using username and password
func Login(username string, password string, deviceName string) (*core.Session, error) {
	return core.Login(username, password, deviceName)
}

// Login to Spotify using an existing authData blob
func LoginSaved(username string, authData []byte, deviceName string) (*core.Session, error) {
	return core.LoginSaved(username, authData, deviceName)
}

// Registers librespot as a Spotify Connect device via mdns. When user connects, logs on to Spotify and saves
// credentials in file at cacheBlobPath. Once saved, the blob credentials allow the program to connect to other
// Spotify Connect devices and control them.
func LoginDiscovery(cacheBlobPath string, deviceName string) (*core.Session, error) {
	return core.LoginDiscovery(cacheBlobPath, deviceName)
}

// Login using an authentication blob through Spotify Connect discovery system, reading an existing blob data. To read
// from a file, see LoginDiscoveryBlobFile.
func LoginDiscoveryBlob(username string, blob string, deviceName string) (*core.Session, error) {
	return core.LoginDiscoveryBlob(username, blob, deviceName)
}

// Login from credentials at cacheBlobPath previously saved by LoginDiscovery. Similar to LoginDiscoveryBlob, except
// it reads it directly from a file.
func LoginDiscoveryBlobFile(cacheBlobPath string, deviceName string) (*core.Session, error) {
	return core.LoginDiscoveryBlobFile(cacheBlobPath, deviceName)
}

// Login to Spotify using the OAuth method
func LoginOAuth(deviceName string, clientId string, clientSecret string) (*core.Session, error) {
	return core.LoginOAuth(deviceName, clientId, clientSecret)
}
