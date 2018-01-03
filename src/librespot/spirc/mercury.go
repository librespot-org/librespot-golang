package spirc

import (
	"Spotify"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"librespot/core"
	"librespot/mercury"
	"net/url"
)

func (c *Controller) mercuryGet(url string) []byte {
	done := make(chan []byte)
	go c.session.Mercury().Request(mercury.Request{
		Method:  "GET",
		Uri:     url,
		Payload: [][]byte{},
	}, func(res mercury.Response) {
		done <- res.CombinePayload()
	})

	result := <-done
	return result
}

func (c *Controller) mercuryGetJson(url string, result interface{}) (err error) {
	data := c.mercuryGet(url)
	err = json.Unmarshal(data, result)
	return
}

func (c *Controller) mercuryGetProto(url string, result proto.Message) (err error) {
	data := c.mercuryGet(url)
	err = proto.Unmarshal(data, result)
	return
}

func (c *Controller) GetRootPlaylist() (*Spotify.SelectedListContent, error) {
	uri := "hm://playlist/user/" + c.session.Username() + "/rootlist"
	result := &Spotify.SelectedListContent{}
	err := c.mercuryGetProto(uri, result)
	return result, err
}

func (c *Controller) GetPlaylist(id string) (*Spotify.SelectedListContent, error) {
	uri := "hm://playlist/" + id

	result := &Spotify.SelectedListContent{}
	err := c.mercuryGetProto(uri, result)
	return result, err
}

func (c *Controller) Search(search string) (*core.SearchResult, error) {
	uri := "hm://searchview/km/v2/search/" + url.QueryEscape(search) + "?limit=12&tracks-limit=100&catalogue=&country=US&locale=en&platform=zelda&username="

	result := &core.SearchResult{}
	err := c.mercuryGetJson(uri, result)
	return result, err
}

func (c *Controller) Suggest(search string) (*core.SuggestResult, error) {
	uri := "hm://searchview/km/v3/suggest/" + url.QueryEscape(search) + "?limit=3&intent=2516516747764520149&sequence=0&catalogue=&country=&locale=&platform=zelda&username="
	data := c.mercuryGet(uri)

	return parseSuggest(data)
}

func (c *Controller) GetTrack(id string) (*Spotify.Track, error) {
	uri := "hm://metadata/3/track/" + id
	result := &Spotify.Track{}
	err := c.mercuryGetProto(uri, result)
	return result, err
}

func parseSuggest(body []byte) (*core.SuggestResult, error) {
	result := &core.SuggestResult{}
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
