package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/mux"
	"github.com/juliotorresmoreno/doppler/config"
	"github.com/juliotorresmoreno/doppler/handler"
	"github.com/juliotorresmoreno/doppler/middleware"
)

func Configure() *http.Server {
	gin.SetMode(gin.DebugMode)
	r := mux.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return middleware.Cors(next)
	})

	conf, _ := config.GetConfig()

	server := &http.Server{
		Addr:         conf.Addr,
		Handler:      handler.ProxyHandler(r),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
	}
	return server
}
