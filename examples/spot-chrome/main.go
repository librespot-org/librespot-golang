package main

import (
	"encoding/base64"
	"github.com/badfortrains/spotcontrol"
	"github.com/gopherjs/gopherjs/js"
)

func setupGlobal() {
	js.Global.Set("spotcontrol", map[string]interface{}{
		"login":      login,
		"loginSaved": loginSaved,
		"convert62":  convert64to62,
	})
}

func convert64to62(data64 string) string {
	data, _ := base64.StdEncoding.DecodeString(data64)
	return spotcontrol.ConvertTo62(data)
}

func loginSaved(username, authData string, appkey string, cb *js.Object) {
	go func() {
		key, _ := base64.StdEncoding.DecodeString(appkey)
		data, _ := base64.StdEncoding.DecodeString(authData)
		conn, _ := MakeConn()
		sController, _, err := spotcontrol.LoginConnectionSaved(username, data, key, "spotcontrol", conn)
		if err != nil {
			cb.Invoke(nil, "", "login failed")
		}
		cb.Invoke(js.MakeWrapper(sController), authData, nil)
	}()
}

func login(username, password, appkey string, cb *js.Object) {
	go func() {
		key, _ := base64.StdEncoding.DecodeString(appkey)
		conn, _ := MakeConn()
		sController, authData, err := spotcontrol.LoginConnection(username, password, key, "spotcontrol", conn)
		if err != nil {
			cb.Invoke(nil, "", "login failed")
		} else {
			cb.Invoke(js.MakeWrapper(sController), base64.StdEncoding.EncodeToString(authData), nil)
		}
	}()
}

func main() {
	setupGlobal()
}
