package main

import (
	"encoding/base64"
	"github.com/badfortrains/spotcontrol"
	"github.com/gopherjs/gopherjs/js"
)

func setupGlobal() {
	js.Global.Set("spotcontrol", map[string]interface{}{
		"login":     login,
		"convert62": convert64to62,
	})
}

func convert64to62(data64 string) string {
	data, _ := base64.StdEncoding.DecodeString(data64)
	return spotcontrol.ConvertTo62(data)
}

func login(username, password, appkey string, cb *js.Object) {
	go func() {
		key, _ := base64.StdEncoding.DecodeString(appkey)
		conn, _ := MakeConn()
		sController := spotcontrol.LoginConnection(username, password, key, "spotcontrol", conn)
		cb.Invoke(js.MakeWrapper(sController))
	}()
}

func main() {
	setupGlobal()
}
