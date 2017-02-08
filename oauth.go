package spotcontrol

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
)

type OAuth struct {
	Access_token  string
	Refresh_token string
	Scope         string
}

func getOAuthToken() OAuth {
	ch := make(chan OAuth)
	clientId := os.Getenv("client_id")
	clientSecret := os.Getenv("client_secret")

	fmt.Println("go to this url")
	urlPath := "https://accounts.spotify.com/authorize?" +
		"client_id=" + clientId +
		"&response_type=code" +
		"&redirect_uri=http://localhost:8888/callback" +
		"&scope=streaming"
	fmt.Println(urlPath)

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()
		fmt.Println("Start callback", params.Get("code"), clientId, clientSecret)

		val := url.Values{}
		val.Set("grant_type", "authorization_code")
		val.Set("code", params.Get("code"))
		val.Set("redirect_uri", "http://localhost:8888/callback")
		val.Set("client_id", clientId)
		val.Set("client_secret", clientSecret)

		resp, err := http.PostForm("https://accounts.spotify.com/api/token", val)
		if err != nil {
			// Retry since there is an nginx bug that causes http2 streams to get
			// an initial REFUSED_STREAM response
			// https://github.com/curl/curl/issues/804
			resp, err = http.PostForm("https://accounts.spotify.com/api/token", val)
			if err != nil {
				return
			}
		}
		defer resp.Body.Close()
		f := OAuth{}
		body, _ := ioutil.ReadAll(resp.Body)
		json.Unmarshal(body, &f)
		if params.Get("code") == "" {
			fmt.Fprintf(w, "failed to get authorization code, visit \n %s", urlPath)
		} else {
			fmt.Fprintf(w, "loggin in...")
			ch <- f
		}
	})

	go func() {
		log.Fatal(http.ListenAndServe(":8888", nil))
	}()

	return <-ch
}
