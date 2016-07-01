package main

import (
	"bytes"
	"fmt"
	"github.com/gopherjs/gopherjs/js"
)

var chromeSocketsTcp = js.Global.Get("chrome").Get("sockets").Get("tcp")

type ChromeTcp struct {
	buffer   *bytes.Buffer
	gotData  chan int
	socketId int
}

func (c *ChromeTcp) listen() {
	chromeSocketsTcp.Get("onReceive").Call("addListener", func(info *js.Object) {
		if info.Get("socketId").Int() != c.socketId {
			return
		}
		go func() {
			bytes := js.Global.Get("Uint8Array").New(info.Get("data")).Interface().([]byte)
			c.buffer.Write(bytes)
			select {
			case c.gotData <- 1:
			default:
			}
		}()
	})

	chromeSocketsTcp.Get("onReceiveError").Call("addListener", func(info *js.Object) {
		if info.Get("socketId").Int() != c.socketId {
			return
		}
		go func() {
			resultCode := info.Get("resultCode").Int()
			c.gotData <- resultCode
		}()
	})
}

func uint8ArrayToArrayBuffer(p *js.Object) *js.Object {
	buffer := p.Get("buffer")
	byteOffset := p.Get("byteOffset").Int()
	byteLength := p.Get("byteLength").Int()
	if byteOffset != 0 || byteLength != buffer.Get("byteLength").Int() {
		return buffer.Call("slice", byteOffset, byteOffset+byteLength)
	}
	return buffer
}

func MakeConn() (*ChromeTcp, error) {
	done := make(chan int)
	buf := make([]byte, 0, 4096)
	conn := &ChromeTcp{
		buffer:  bytes.NewBuffer(buf),
		gotData: make(chan int),
	}
	config := &js.Object{}
	chromeSocketsTcp.Call("create", config, func(createInfo *js.Object) {
		conn.socketId = createInfo.Get("socketId").Int()
		chromeSocketsTcp.Call("connect", createInfo.Get("socketId").Int(),
			"lon3-accesspoint-a57.ap.spotify.com", 4070, func(result *js.Object) {
				done <- result.Int()
			})
	})

	res := <-done
	if res < 0 {
		return nil, fmt.Errorf("Failed chrome.sockets.tcp.connect, error code %v", res)
	}
	conn.listen()
	return conn, nil
}

func (c *ChromeTcp) Write(buf []byte) (int, error) {
	done := make(chan int)
	arrayBuffer := js.InternalObject(uint8ArrayToArrayBuffer).Invoke(buf)
	chromeSocketsTcp.Call("send", c.socketId, arrayBuffer, func(bytesWritten *js.Object) {
		done <- bytesWritten.Int()
	})

	res := <-done
	if res >= 0 {
		return res, nil
	} else {
		return 0, fmt.Errorf("Failed chrome.sockets.tcp write, error code: %v", res)
	}

}

func (c *ChromeTcp) Read(buf []byte) (int, error) {
	if c.buffer.Len() == 0 {
		resultCode := <-c.gotData
		if resultCode < 0 {
			return 0, fmt.Errorf("Failed chrome.sockets.tcp read, error code: %v", resultCode)
		}
	}
	return c.buffer.Read(buf)
}
