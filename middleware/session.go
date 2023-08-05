package middleware

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/juliotorresmoreno/doppler/db"
)

func Session(ctx *gin.Context) {
	token := ctx.Request.Header.Get("Authorization")
	if !strings.HasPrefix(token, "Bearer") {
		token = ctx.Request.URL.Query().Get("token")
	}
	if token == "" {
		return
	}

	redisCli := db.GetRedisClient()
	result := redisCli.Get(token)

	fmt.Println(result.Val())
}
