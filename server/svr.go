package server

import (
	"crypto/tls"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/juliotorresmoreno/doppler/config"
	"github.com/juliotorresmoreno/doppler/handler"
)

func Configure() *http.Server {
	gin.SetMode(gin.DebugMode)
	r := gin.Default()
	tls.Client(&net.TCPConn{}, &tls.Config{})

	handler.AttachStatic(r)
	conf, _ := config.GetConfig()

	server := &http.Server{
		Addr:    conf.Addr,
		Handler: handler.ProxyHandler(r),
	}
	return server
}
