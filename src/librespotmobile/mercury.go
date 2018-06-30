package librespotmobile

import (
	"encoding/json"
	"librespot/core"
	"librespot/mercury"
)

type MobileMercury struct {
	mercury *mercury.Client
}

func marshalJson(obj interface{}) string {
	data, _ := json.Marshal(obj)
	return string(data)
}

func createMobileMercury(session *core.Session) *MobileMercury {
	return &MobileMercury{
		mercury: session.Mercury(),
	}
}

func (m *MobileMercury) GetTrack(id string) (string, error) {
	spt, err := m.mercury.GetTrack(id)
	if err != nil {
		return "", err
	}

	return marshalJson(spt), nil
}

func (m *MobileMercury) GetAlbum(id string) (string, error) {
	spt, err := m.mercury.GetAlbum(id)
	if err != nil {
		return "", err
	}

	return marshalJson(spt), nil
}

func (m *MobileMercury) GetArtist(id string) (string, error) {
	spt, err := m.mercury.GetArtist(id)
	if err != nil {
		return "", err
	}

	return marshalJson(spt), nil
}

func (m *MobileMercury) GetPlaylist(id string) (string, error) {
	spt, err := m.mercury.GetPlaylist(id)
	if err != nil {
		return "", err
	}

	return marshalJson(spt), nil
}

func (m *MobileMercury) GetRootPlaylist(username string) (string, error) {
	spt, err := m.mercury.GetRootPlaylist(username)
	if err != nil {
		return "", err
	}

	return marshalJson(spt), nil
}

func (m *MobileMercury) GetToken(clientId string, scopes string) (string, error) {
	spt, err := m.mercury.GetToken(clientId, scopes)
	if err != nil {
		return "", err
	}

	return marshalJson(spt), nil
}
