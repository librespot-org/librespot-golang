package metadata

import (
	"encoding/json"
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

type Playlist struct {
	Name           string `json:"name"`
	Uri            string `json:"uri"`
	Image          string `json:"image"`
	FollowersCount int    `json:"followersCount"`
	Author         string `json:"author"`
}

type Profile struct {
	Name           string `json:"name"`
	Uri            string `json:"uri"`
	Image          string `json:"image"`
	FollowersCount int    `json:"followersCount"`
}

type Genre struct {
	Name  string `json:"name"`
	Uri   string `json:"uri"`
	Image string `json:"image"`
}

type TopHit struct {
	Uri            string `json:"uri"`
	Name           string `json:"name"`
	Image          string `json:"image"`
	Verified       bool   `json:"verified"`
	Following      bool   `json:"following"`
	FollowersCount int    `json:"followersCount"`
	Author         string `json:"author"`
	Log            struct {
		Origin string `json:"origin"`
		TopHit string `json:"top_hit"`
	} `json:"log"`
	Artists []Artist `json:"artists"`
	Album   Album    `json:"album"`
}

type Show struct {
	Name     string `json:"name"`
	Uri      string `json:"uri"`
	Image    string `json:"image"`
	ShowType string `json:"showType"`
}

type VideoEpisode struct {
	Name  string `json:"name"`
	Uri   string `json:"uri"`
	Image string `json:"image"`
}

type TopRecommendation struct {
}

type SearchResponse struct {
	Results         SearchResult `json:"results"`
	RequestId       string       `json:"requestId"`
	CategoriesOrder []string     `json:"categoriesOrder"`
}

type SearchResult struct {
	Tracks struct {
		Hits  []Track `json:"hits"`
		Total int     `json:"total"`
	} `json:"tracks"`

	Albums struct {
		Hits  []Album `json:"hits"`
		Total int     `json:"total"`
	} `json:"albums"`

	Artists struct {
		Hits  []Artist `json:"hits"`
		Total int      `json:"total"`
	} `json:"artists"`

	Playlists struct {
		Hits  []Playlist `json:"hits"`
		Total int        `json:"total"`
	} `json:"playlists"`

	Profiles struct {
		Hits  []Profile `json:"hits"`
		Total int       `json:"total"`
	} `json:"profiles"`

	Genres struct {
		Hits  []Genre `json:"hits"`
		Total int     `json:"total"`
	} `json:"genres"`

	TopHit struct {
		Hits  []TopHit `json:"hits"`
		Total int      `json:"total"`
	} `json:"topHit"`

	Shows struct {
		Hits  []Show `json:"hits"`
		Total int    `json:"total"`
	} `json:"shows"`

	VideoEpisodes struct {
		Hits  []VideoEpisode `json:"hits"`
		Total int            `json:"total"`
	} `json:"videoEpisodes"`

	TopRecommendations struct {
		Hits  []TopRecommendation `json:"hits"`
		Total int                 `json:"total"`
	} `json:"topRecommendations"`

	Error error
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

type Token struct {
	AccessToken string   `json:"accessToken"`
	ExpiresIn   int      `json:"expiresIn"`
	TokenType   string   `json:"tokenType"`
	Scope       []string `json:"scope"`
}
