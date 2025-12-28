package server

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/juliotorresmoreno/doppler/config"
	"github.com/juliotorresmoreno/doppler/handler"
	"github.com/juliotorresmoreno/doppler/middleware"
)

func Configure() *http.Server {
	r := mux.NewRouter()
	r.Use(middleware.Cors)

	conf, _ := config.GetConfig()

	server := &http.Server{
		Addr:         conf.Addr,
		Handler:      handler.ProxyHandler(r),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
	}
	return server
}
