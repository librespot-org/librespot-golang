package mercury

import (
	"Spotify"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
	"librespot/metadata"
	"net/url"
)

func (m *Client) mercuryGet(url string) []byte {
	done := make(chan []byte)
	go m.Request(Request{
		Method:  "GET",
		Uri:     url,
		Payload: [][]byte{},
	}, func(res Response) {
		done <- res.CombinePayload()
	})

	result := <-done
	return result
}

func (m *Client) mercuryGetJson(url string, result interface{}) (err error) {
	data := m.mercuryGet(url)
	err = json.Unmarshal(data, result)
	return
}

func (m *Client) mercuryGetProto(url string, result proto.Message) (err error) {
	data := m.mercuryGet(url)
	// ioutil.WriteFile("/tmp/proto.blob", data, 0644)
	err = proto.Unmarshal(data, result)
	return
}

func (m *Client) GetRootPlaylist(username string) (*Spotify.SelectedListContent, error) {
	uri := "hm://playlist/user/" + username + "/rootlist"
	result := &Spotify.SelectedListContent{}
	err := m.mercuryGetProto(uri, result)
	return result, err
}

func (m *Client) GetPlaylist(id string) (*Spotify.SelectedListContent, error) {
	uri := "hm://playlist/" + id

	result := &Spotify.SelectedListContent{}
	err := m.mercuryGetProto(uri, result)
	return result, err
}

func (m *Client) Search(search string) (*metadata.SearchResult, error) {
	uri := "hm://searchview/km/v2/search/" + url.QueryEscape(search) + "?limit=12&tracks-limit=100&catalogue=&country=US&locale=en&platform=zelda&username="

	result := &metadata.SearchResult{}
	err := m.mercuryGetJson(uri, result)
	return result, err
}

func (m *Client) Suggest(search string) (*metadata.SuggestResult, error) {
	uri := "hm://searchview/km/v3/suggest/" + url.QueryEscape(search) + "?limit=3&intent=2516516747764520149&sequence=0&catalogue=&country=&locale=&platform=zelda&username="
	data := m.mercuryGet(uri)

	return parseSuggest(data)
}

func (m *Client) GetTrack(id string) (*Spotify.Track, error) {
	uri := "hm://metadata/4/track/" + id
	result := &Spotify.Track{}
	err := m.mercuryGetProto(uri, result)
	return result, err
}

func (m *Client) GetArtist(id string) (*Spotify.Artist, error) {
	uri := "hm://metadata/4/artist/" + id
	result := &Spotify.Artist{}
	err := m.mercuryGetProto(uri, result)
	return result, err
}

func (m *Client) GetAlbum(id string) (*Spotify.Album, error) {
	uri := "hm://metadata/4/album/" + id
	result := &Spotify.Album{}
	err := m.mercuryGetProto(uri, result)
	return result, err
}

func parseSuggest(body []byte) (*metadata.SuggestResult, error) {
	result := &metadata.SuggestResult{}
	err := json.Unmarshal(body, result)
	if err != nil {
		fmt.Println("err", err)
	}

	for _, s := range result.Sections {
		switch s.Typ {
		case "top-results":
			err = json.Unmarshal(s.RawItems, &result.TopHits)
		case "album-results":
			err = json.Unmarshal(s.RawItems, &result.Albums)
		case "artist-results":
			err = json.Unmarshal(s.RawItems, &result.Artists)
		case "track-results":
			err = json.Unmarshal(s.RawItems, &result.Tracks)
		}
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
