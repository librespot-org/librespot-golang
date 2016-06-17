package spotcontrol

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"math/rand"
	"net/http"
)

type aplist struct {
	ApList []string `json:"ap_list"`
}

func apresolve() (string, error) {
	apendpoint := "http://apresolve.spotify.com/"
	r, err := http.Get(apendpoint)
	if err != nil {
		return "", err
	}
	defer r.Body.Close()

	endpoints := &aplist{}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(body, endpoints)
	if err != nil {
		return "", err
	}
	if len(endpoints.ApList) == 0 {
		return "", errors.New("Ap enpoint list is empty")
	}

	return endpoints.ApList[rand.Intn(len(endpoints.ApList))], nil
}
