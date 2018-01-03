package core

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
