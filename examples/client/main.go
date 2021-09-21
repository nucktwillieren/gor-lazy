package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"

	//"encoding/hex"
	"flag"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "127.0.0.1:3000", "http service address")

func main() {
	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}

	headers := make(http.Header)
	headers.Set("Authorization", "test-client")
	headers.Set("Sec-WebSocket-Protocol", "protocol-1")

	dialer := websocket.Dialer{TLSClientConfig: &tls.Config{RootCAs: nil, InsecureSkipVerify: true}}
	c, _, err := dialer.Dial(u.String(), headers)
	if err != nil {
		log.Println("dial:", err)
		return
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			if c == nil {
				return
			}
			_, msg, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			fmt.Print(string(msg), ",")
		}
	}()

	t := map[string]interface{}{
		"type":     "device",
		"method":   "action-build",
		"ClientID": "12345",
		"data": map[string]string{
			"building": "mining_factory",
		},
	}

	d, _ := json.Marshal(t)

	for {
		if c == nil {
			return
		}
		c.WriteMessage(1, d)
		time.Sleep(100 * time.Millisecond)
	}
}
