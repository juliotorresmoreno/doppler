package middleware

import (
	"github.com/gin-gonic/gin"
)

func Cors(ctx *gin.Context) {
	origin := ctx.Request.Header.Get("Origin")
	if origin == "" {
		origin = "*"
	}

	ctx.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	ctx.Writer.Header().Set("Access-Control-Allow-Origin", origin)
	ctx.Writer.Header().Set("Access-Control-Allow-Methods", "GET,PUT,POST,DELETE,OPTIONS")
	ctx.Writer.Header().Set("Access-Control-Allow-Headers", "X-Requested-With,Content-Type,Cache-Control")

	if ctx.Request.Method == "OPTIONS" {
		ctx.Writer.WriteHeader(200)
		ctx.Writer.Flush()
		return
	}
}
