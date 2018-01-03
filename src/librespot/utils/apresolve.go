package utils

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"math/rand"
	"net/http"
)

const kAPEndpoint = "https://APResolve.spotify.com/"

// APList is the JSON structure corresponding to the output of the AP endpoint resolve API
type APList struct {
	ApList []string `json:"ap_list"`
}

// APResolve fetches the available Spotify servers (AP) and picks a random one
func APResolve() (string, error) {
	r, err := http.Get(kAPEndpoint)
	if err != nil {
		return "", err
	}
	defer r.Body.Close()

	var endpoints APList

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(body, &endpoints)
	if err != nil {
		return "", err
	}
	if len(endpoints.ApList) == 0 {
		return "", errors.New("AP endpoint list is empty")
	}

	return endpoints.ApList[rand.Intn(len(endpoints.ApList))], nil
}
