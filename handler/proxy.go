package handler

import (
	"io"
	"net"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/juliotorresmoreno/doppler/internal/utils"
)

type Proxy struct {
	Fallback http.Handler
	Logger   utils.Logger
}

func ProxyHandler(fallback http.Handler) http.Handler {
	p := &Proxy{Fallback: fallback, Logger: &utils.LoggerBD{}}
	return http.HandlerFunc(p.Handle)
}

func (h *Proxy) Handle(res http.ResponseWriter, req *http.Request) {
	isRequestToProxy, protocol := utils.IsRequestToProxy(req)
	if isRequestToProxy {
		if protocol == "http" {
			h.HandleHTTP(res, req)
			return
		}
		if protocol == "connect" {
			h.HandleConnect(res, req)
			return
		}
	}

	h.Fallback.ServeHTTP(res, req)
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

	h.Logger.Register(req.Method, resp.StatusCode, w.Header(), nil)
}

func (h *Proxy) HandleConnect(w http.ResponseWriter, r *http.Request) {
	var sw http.ResponseWriter
	metaValue := reflect.ValueOf(w).Elem()
	field := metaValue.FieldByName("ResponseWriter")
	if field.IsValid() && field.Interface() != nil {
		value := field.Interface()
		sw = value.(http.ResponseWriter)
	} else {
		sw = w
	}

	destConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		http.Error(sw, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	hijacker, ok := sw.(http.Hijacker)
	if !ok {
		http.Error(sw, "Hijacking not supported", http.StatusInternalServerError)
		h.Logger.Register("CONNECT", http.StatusInternalServerError, w.Header(), nil)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(sw, err.Error(), http.StatusServiceUnavailable)
		h.Logger.Register("CONNECT", http.StatusInternalServerError, w.Header(), nil)
		return
	}

	wg := sync.WaitGroup{}

	wg.Add(2)
	go func() {
		utils.Transfer(destConn, clientConn)
		wg.Done()
	}()
	go func() {
		utils.Transfer(clientConn, destConn)
		wg.Done()
	}()
	wg.Wait()

	h.Logger.Register("CONNECT", http.StatusOK, w.Header(), nil)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
