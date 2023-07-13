package client

import (
	"bytes"
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type Action struct {
	Command string
	Payload map[string]interface{}
	Client  *Client
}

func Handler(c *Client, done chan struct{}) {
	defer close(done)
	for {
		_, data, err := c.Conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", data)

		message := map[string]interface{}{}
		json.NewDecoder(bytes.NewBuffer(data)).Decode(&message)
		if _, ok := message["command"]; ok {
			c.command <- &Action{
				Command: message["command"].(string),
				Payload: message["payload"].(map[string]interface{}),
				Client:  c,
			}
		}
	}
	done <- struct{}{}
}

func Alive(c *Client, done chan struct{}) {
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case t := <-ticker.C:
			err := c.Conn.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				log.Println("write:", err)
				return
			}
		}
	}
}
