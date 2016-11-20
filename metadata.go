package spotcontrol

import (
	"encoding/json"
	"fmt"
	Spotify "github.com/badfortrains/spotcontrol/proto"
	"github.com/golang/protobuf/proto"
	"net/url"
)

type Artist struct {
	Image string `json:"image"`
	Name  string `json:"name"`
	Uri   string `json:"uri"`
}

type Album struct {
	Artists []Artist `json:"artists"`
	Image   string   `json:"image"`
	Name    string   `json:"name"`
	Uri     string   `json:"uri"`
}

type Track struct {
	Album      Album    `json:"album"`
	Artists    []Artist `json:"artists"`
	Image      string   `json:"image"`
	Name       string   `json:"name"`
	Uri        string   `json:"uri"`
	Duration   int      `json:"duration"`
	Popularity int      `json:"popularity"`
}

type TopHit struct {
	Image string `json:"image"`
	Name  string `json:"name"`
	Uri   string `json:"uri"`
	Log   struct {
		Origin string `json:"origin"`
		TopHit string `json:"top_hit"`
	} `json:"log"`
	Artists []Artist `json:"artists"`
	Album   Album    `json:"album"`
}

type SearchResult struct {
	Artists struct {
		Hits  []Artist `json:"hits"`
		Total int      `json:"total"`
	} `json:"artists"`
	Albums struct {
		Hits  []Album `json:"hits"`
		Total int     `json:"total"`
	} `json:"albums"`
	Tracks struct {
		Hits  []Track `json:"hits"`
		Total int     `json:"total"`
	} `json:"tracks"`
	Error error
}

func (c *SpircController) mercuryGet(url string) []byte {
	done := make(chan []byte)
	go c.session.mercurySendRequest(mercuryRequest{
		method:  "GET",
		uri:     url,
		payload: [][]byte{},
	}, func(res mercuryResponse) {
		done <- res.combinePayload()
	})

	result := <-done
	return result
}

func (c *SpircController) mercuryGetJson(url string, result interface{}) (err error) {
	data := c.mercuryGet(url)
	err = json.Unmarshal(data, result)
	return
}

func (c *SpircController) mercuryGetProto(url string, result proto.Message) (err error) {
	data := c.mercuryGet(url)
	err = proto.Unmarshal(data, result)
	return
}

type SuggestResult struct {
	Sections []struct {
		RawItems json.RawMessage `json:"items"`
		Typ      string          `json:"type"`
	} `json:"sections"`
	Albums  []Artist
	Artists []Album
	Tracks  []Track
	TopHits []TopHit
	Error   error
}

func parseSuggest(body []byte) (*SuggestResult, error) {
	result := &SuggestResult{}
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

func (res *mercuryResponse) combinePayload() []byte {
	body := make([]byte, 0)
	for _, p := range res.payload {
		body = append(body, p...)
	}
	return body
}

func (c *SpircController) GetRootPlaylist() (*Spotify.SelectedListContent, error) {
	url := "hm://playlist/user/" + c.username + "/rootlist"
	result := &Spotify.SelectedListContent{}
	err := c.mercuryGetProto(url, result)
	return result, err
}

func (c *SpircController) GetPlaylist(id string) (*Spotify.SelectedListContent, error) {
	url := "hm://playlist/" + id

	result := &Spotify.SelectedListContent{}
	err := c.mercuryGetProto(url, result)
	return result, err
}

func (c *SpircController) Search(search string) (*SearchResult, error) {
	url := "hm://searchview/km/v2/search/" + url.QueryEscape(search) + "?limit=12&tracks-limit=100&catalogue=&country=US&locale=en&platform=zelda&username="

	result := &SearchResult{}
	err := c.mercuryGetJson(url, result)
	return result, err
}

func (c *SpircController) Suggest(search string) (*SuggestResult, error) {
	url := "hm://searchview/km/v3/suggest/" + url.QueryEscape(search) + "?limit=3&intent=2516516747764520149&sequence=0&catalogue=&country=&locale=&platform=zelda&username="
	data := c.mercuryGet(url)

	return parseSuggest(data)
}

func (c *SpircController) GetTrack(id string) (*Spotify.Track, error) {
	url := "hm://metadata/3/track/" + id
	result := &Spotify.Track{}
	err := c.mercuryGetProto(url, result)
	return result, err
}
