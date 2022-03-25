package mercury

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/golang/protobuf/proto"
	"github.com/librespot-org/librespot-golang/Spotify"
	"github.com/librespot-org/librespot-golang/librespot/metadata"
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
	// fmt.Printf("%s", data)
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
	uri := fmt.Sprintf("hm://playlist/user/%s/rootlist", username)

	result := &Spotify.SelectedListContent{}
	err := m.mercuryGetProto(uri, result)
	return result, err
}

func (m *Client) GetPlaylist(id string) (*Spotify.SelectedListContent, error) {
	uri := fmt.Sprintf("hm://playlist/%s", id)

	result := &Spotify.SelectedListContent{}
	err := m.mercuryGetProto(uri, result)
	return result, err
}

func (m *Client) GetToken(clientId string, scopes string) (*metadata.Token, error) {
	uri := fmt.Sprintf("hm://keymaster/token/authenticated?client_id=%s&scope=%s", url.QueryEscape(clientId),
		url.QueryEscape(scopes))

	token := &metadata.Token{}
	err := m.mercuryGetJson(uri, token)
	return token, err
}

func (m *Client) Search(search string, limit int, country string, username string) (*metadata.SearchResponse, error) {
	v := url.Values{}
	v.Set("entityVersion", "2")
	v.Set("limit", fmt.Sprintf("%d", limit))
	v.Set("imageSize", "large")
	v.Set("catalogue", "")
	v.Set("country", country)
	v.Set("platform", "zelda")
	v.Set("username", username)

	uri := fmt.Sprintf("hm://searchview/km/v4/search/%s?%s", url.QueryEscape(search), v.Encode())

	result := &metadata.SearchResponse{}
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

func (m *Client) GetEpisode(id string) (*Spotify.Episode, error) {
	uri := "hm://metadata/3/episode/" + id
	result := &Spotify.Episode{}
	err := m.mercuryGetProto(uri, result)
	return result, err
}

func (m *Client) GetShow(id string) (*Spotify.Show, error) {
	uri := "hm://metadata/3/show/" + id
	result := &Spotify.Show{}
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
