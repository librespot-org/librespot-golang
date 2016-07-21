// +build js spotandroid

package spotcontrol

import (
	"encoding/json"
	"fmt"
	Spotify "github.com/badfortrains/spotcontrol/proto"
	"io"
)

type Updater interface {
	OnUpdate(device string)
}

func LoginConnection(username string, password string,
	appkey []byte, deviceName string, con io.ReadWriter) (*SpircController, error) {
	s := &session{
		keys:               generateKeys(),
		tcpCon:             con,
		mercuryConstructor: setupMercury,
		shannonConstructor: setupStream,
	}
	s.deviceId = generateDeviceId(deviceName)
	s.deviceName = deviceName

	s.startConnection()
	loginPacket := loginPacketPassword(appkey, username, password, s.deviceId)
	return s.doLogin(loginPacket, username)
}

func LoginConnectionSaved(username string, authData []byte,
	appkey []byte, deviceName string, con io.ReadWriter) (*SpircController, error) {
	s := &session{
		keys:               generateKeys(),
		tcpCon:             con,
		mercuryConstructor: setupMercury,
		shannonConstructor: setupStream,
	}
	s.deviceId = generateDeviceId(deviceName)
	s.deviceName = deviceName

	s.startConnection()
	packet := loginPacket(appkey, username, authData,
		Spotify.AuthenticationType_AUTHENTICATION_STORED_SPOTIFY_CREDENTIALS.Enum(), s.deviceId)
	return s.doLogin(packet, username)
}

func (c *SpircController) HandleUpdatesCb(cb func(device string)) {
	c.updateChan = make(chan Spotify.Frame, 5)

	go func() {
		for {
			update := <-c.updateChan
			json, err := json.Marshal(update)
			if err != nil {
				fmt.Println("Error marhsaling device json")
			} else {
				cb(string(json))
			}
		}
	}()
}

func (c *SpircController) HandleUpdates(u Updater) {
	c.updateChan = make(chan Spotify.Frame, 5)

	go func() {
		for {
			update := <-c.updateChan
			json, err := json.Marshal(update)
			if err != nil {
				fmt.Println("Error marhsaling device json")
			} else {
				u.OnUpdate(string(json))
			}
		}
	}()
}

func (c *SpircController) ListDevicesJson() (string, error) {
	devices := c.ListDevices()
	json, err := json.Marshal(devices)
	if err != nil {
		return "", nil
	}
	return string(json), nil
}

func (c *SpircController) ListMdnsDevicesJson() (string, error) {
	devices := c.ListMdnsDevices()
	json, err := json.Marshal(devices)
	if err != nil {
		return "", nil
	}
	return string(json), nil
}

func (c *SpircController) SuggestJson(term string) (string, error) {
	result, err := c.Suggest(term)
	if err != nil {
		return "", nil
	}
	json, err := json.Marshal(result)
	if err != nil {
		return "", nil
	}
	return string(json), nil
}

func (c *SpircController) SearchJson(term string) (string, error) {
	result, err := c.Search(term)
	if err != nil {
		return "", nil
	}
	json, err := json.Marshal(result)
	if err != nil {
		return "", nil
	}
	return string(json), nil
}
