package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/juliotorresmoreno/doppler/helper"
	"github.com/juliotorresmoreno/doppler/internal/utils"
)

var sockets = sync.Map{}
var clients = sync.Map{}

type Client struct {
	Conn     *websocket.Conn
	Protocol string
	Domain   string
	Reverse  string
	Mutex    *sync.Mutex
	handlers map[string]func(data interface{})
}

func (c *Client) RegisterHandler(id string, fn func(data interface{})) {
	c.handlers[id] = fn
}

type Action struct {
	Command string
	Payload map[string]interface{}
	Conn    *websocket.Conn
}

type Proxy struct {
	Fallback http.Handler
	Logger   utils.Logger
	Upgrader *websocket.Upgrader
	command  chan *Action
}

func ProxyHandler(fallback http.Handler) http.Handler {
	var upgrader = &websocket.Upgrader{
		HandshakeTimeout:  1 * time.Minute,
		EnableCompression: true,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	p := &Proxy{
		Fallback: fallback,
		Logger:   &utils.LoggerBD{},
		Upgrader: upgrader,
		command:  make(chan *Action),
	}
	go p.HandleCMD()

	return http.HandlerFunc(p.Handle)
}

func (h *Proxy) Handle(res http.ResponseWriter, req *http.Request) {
	isRequestToProxy, protocol := utils.IsRequestToProxy(req)
	if !isRequestToProxy {
		if _, ok := clients.Load(req.Header.Get("Host")); ok {
			h.HandleReverseHTTP(res, req, req.Header.Get("Host"))
			return
		}
		h.Fallback.ServeHTTP(res, req)
		return
	}

	// credentials := req.Header.Get("Authorization")
	// if len(credentials) < 6 {
	// 	credentials = req.Header.Get("Proxy-Authorization")
	// }

	// if len(credentials) > 6 {
	// 	if err := h.BasicAuth(credentials); err != nil {
	// 		h.AuthRequired(res, req)
	// 		return
	// 	}
	// }

	switch protocol {
	case "http":
		if _, ok := clients.Load(req.URL.Host); ok {
			h.HandleReverseHTTP(res, req, req.URL.Host)
			return
		}
		h.HandleHTTP(res, req)
	case "websocket":
		h.HandleExpose(res, req)
	case "connect":

		h.HandleConnect(res, req)
	}
}

func (h *Proxy) HandleExpose(res http.ResponseWriter, req *http.Request) {
	ws, err := h.Upgrader.Upgrade(res, req, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer ws.Close()
	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			break
		}
		message := map[string]interface{}{}
		json.NewDecoder(bytes.NewBuffer(data)).Decode(&message)
		if _, ok := message["command"]; ok {
			h.command <- &Action{
				Command: message["command"].(string),
				Payload: message["payload"].(map[string]interface{}),
				Conn:    ws,
			}
		}
	}
	h.command <- &Action{
		Command: "close",
		Conn:    ws,
	}
}

func (h *Proxy) HandleReverseHTTP(res http.ResponseWriter, req *http.Request, alias string) {
	_client, _ := clients.Load(alias)
	client := _client.(*Client)
	client.Mutex.Lock()
	defer client.Mutex.Unlock()
	buff := bytes.NewBufferString("")
	io.Copy(buff, req.Body)
	data := make([]byte, base64.StdEncoding.EncodedLen(buff.Len()))

	base64.StdEncoding.Encode(data, buff.Bytes())
	client.Conn.WriteJSON(map[string]interface{}{
		"command": "request",
		"payload": map[string]interface{}{
			"uuid":   uuid.New(),
			"method": req.Method,
			"url":    client.Protocol + "://" + client.Reverse + req.URL.Path,
			"header": req.Header,
			"body":   string(data),
		},
	})
	time.Sleep(time.Minute)
}

func (h *Proxy) HandleCMD() {
	for {
		action := <-h.command

		switch action.Command {
		case "register":
			payload := action.Payload
			alias := payload["alias"].(string)
			expose := payload["expose"].(string)
			protocol := payload["protocol"].(string)

			if _, ok := clients.Load(alias); ok {
				action.Conn.WriteJSON(map[string]interface{}{
					"event": "unauthorized",
				})
				action.Conn.Close()
				continue
			}

			sockets.Store(action.Conn, alias)
			func() {
				mutex := &sync.Mutex{}
				mutex.Lock()
				defer mutex.Unlock()

				client, _ := clients.LoadOrStore(alias, &Client{
					Conn:     action.Conn,
					Domain:   alias,
					Reverse:  expose,
					Protocol: protocol,
					Mutex:    mutex,
				})
				client.(*Client).Conn.WriteJSON(map[string]interface{}{
					"event": "registered",
				})
			}()
		case "close":
			alias, ok := sockets.LoadAndDelete(action.Conn)
			if ok {
				clients.Delete(alias)
			}
		}
	}
}

func (h *Proxy) HandleHTTP(w http.ResponseWriter, req *http.Request) {
	req.Header.Del("Proxy-Authorization")
	req.Header.Del("Proxy-Connection")
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	io.Copy(w, resp.Body)

	h.Logger.Register(req.Host, req.Method, resp.StatusCode)
}

func (h *Proxy) HandleConnect(w http.ResponseWriter, r *http.Request) {
	destConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	clientConn, err := helper.GetHijack(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		h.Logger.Register(r.Host, "CONNECT", http.StatusInternalServerError)
		return
	}

	go io.Copy(destConn, clientConn)
	go utils.CopyWithRateLimit(clientConn, destConn, 100)

	h.Logger.Register(r.Host, "CONNECT", http.StatusOK)
}

const authenticationRequiredHTML = `
<!DOCTYPE HTML "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>407 Proxy Authentication Required</title>
</head><body>
<h1>Proxy Authentication Required</h1>
<p>This server could not verify that you
are authorized to access the document
requested.  Either you supplied the wrong
credentials (e.g., bad password), or your
browser doesn't understand how to supply
the credentials required.</p>
</body></html>
`

const ACLDeniedHTML = `
<!DOCTYPE HTML "-//IETF//DTD HTML 2.0//EN">
<html><head>
<title>401 Proxy Authentication Denied</title>
</head><body>
<h1>Proxy Authentication Denied</h1>
<p>You do not have permission to access this site.</p>
</body></html>
`

func (h *Proxy) AuthRequired(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Proxy-Authenticate", "Basic realm=\"Doppler\"")
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusProxyAuthRequired)
	w.Write([]byte(authenticationRequiredHTML))
}

// basicAuth .
func (h *Proxy) BasicAuth(credentials string) error {
	if len(credentials) < 6 {
		return errors.New("Unauthorized")
	}
	decoded, err := base64.StdEncoding.DecodeString(credentials[6:])
	if err != nil {
		return errors.New("Unauthorized")
	}
	splitData := strings.Split(string(decoded), ":")
	if len(splitData) == 1 {
		return errors.New("Unauthorized")
	}
	username := splitData[0]
	password := splitData[1]

	// TODO: Validate username and password
	if username == "admin" && password == "admin" {
		return nil
	}

	return errors.New("Unauthorized")
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
