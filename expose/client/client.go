package client

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn    *websocket.Conn
	Mutex   *sync.Mutex
	command chan *Action
}

func (c *Client) Prompt() {
	for {
		cmd := <-c.command

		switch cmd.Command {
		case "request":
			payload := cmd.Payload
			method := payload["method"].(string)
			url := payload["url"].(string)
			body := payload["body"].(string)
			data := make([]byte, base64.StdEncoding.DecodedLen(len(body)))

			base64.StdEncoding.Decode(data, []byte(body))
			req, err := http.NewRequest(method, url, bytes.NewBuffer(data))

			if err != nil {
				fmt.Println(err)
				continue
			}
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println(err)
				continue
			}
			buff := bytes.NewBufferString("")
			io.Copy(buff, res.Body)
			os.Stdout.Write(buff.Bytes())
		}
	}
}

func Reverse(addr, alias, expose, protocol string) {
	u := url.URL{Scheme: "ws", Host: addr, Path: "/echo?nada=1"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	client := &Client{
		Conn:    c,
		Mutex:   &sync.Mutex{},
		command: make(chan *Action),
	}
	go client.Prompt()
	done := make(chan struct{})
	go Handler(client, done)

	c.WriteJSON(map[string]interface{}{
		"command": "register",
		"payload": map[string]interface{}{
			"alias":    alias,
			"expose":   expose,
			"protocol": protocol,
		},
	})

	Alive(client, done)
}
