package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

type OAuth struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	Error        string
}

func GetOauthAccessToken(code string, redirectUri string, clientId string, clientSecret string) (*OAuth, error) {
	val := url.Values{}
	val.Set("grant_type", "authorization_code")
	val.Set("code", code)
	val.Set("redirect_uri", redirectUri)
	val.Set("client_id", clientId)
	val.Set("client_secret", clientSecret)

	resp, err := http.PostForm("https://accounts.spotify.com/api/token", val)
	if err != nil {
		// Retry since there is an nginx bug that causes http2 streams to get
		// an initial REFUSED_STREAM response
		// https://github.com/curl/curl/issues/804
		resp, err = http.PostForm("https://accounts.spotify.com/api/token", val)
		if err != nil {
			return nil, err
		}
	}
	defer resp.Body.Close()
	auth := OAuth{}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &auth)
	if err != nil {
		return nil, err
	}
	if auth.Error != "" {
		return nil, fmt.Errorf("error getting token %v", auth.Error)
	}
	return &auth, nil
}

func getOAuthToken(clientId string, clientSecret string) OAuth {
	ch := make(chan OAuth)

	fmt.Println("go to this url")
	urlPath := "https://accounts.spotify.com/authorize?" +
		"client_id=" + clientId +
		"&response_type=code" +
		"&redirect_uri=http://localhost:8888/callback" +
		"&scope=streaming"
	fmt.Println(urlPath)

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()
		auth, err := GetOauthAccessToken(params.Get("code"), "http://localhost:8888/callback", clientId, clientSecret)
		if err != nil {
			fmt.Fprintf(w, "Error getting token %q", err)
			return
		}
		fmt.Fprintf(w, "Got token, loggin in")
		ch <- *auth
	})

	go func() {
		log.Fatal(http.ListenAndServe(":8888", nil))
	}()

	return <-ch
}
