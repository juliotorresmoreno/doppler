package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/juliotorresmoreno/doppler/db"
	"github.com/juliotorresmoreno/doppler/handler"
	"github.com/juliotorresmoreno/doppler/model"
)

func Session(ctx *gin.Context) {
	token := ctx.Request.Header.Get("Authorization")
	if !strings.HasPrefix(token, "Bearer") {
		token = ctx.Request.URL.Query().Get("token")
	} else {
		token = token[7:]
	}
	if token == "" {
		return
	}

	conn, err := db.GetConnection()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, &handler.ResponseError{
			Code: handler.StatusInternalServerErrorMessage,
		})
		return
	}

	redisCli := db.GetRedisClient()
	result := redisCli.Get(token)

	userId, _ := strconv.Atoi(result.Val())
	user := &model.User{Id: uint(userId)}
	tx := conn.Find(user)

	if tx.Error != nil {
		ctx.JSON(http.StatusInternalServerError, &handler.ResponseError{
			Code: handler.StatusInternalServerErrorMessage,
		})
		return
	}
	session := &handler.Session{
		User: &handler.User{
			Id:       user.Id,
			Name:     user.Name,
			Lastname: user.Lastname,
			Email:    user.Email,
		},
		Token: token,
	}
	ctx.Set("session", session)
}
