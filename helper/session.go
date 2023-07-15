package helper

import (
	"time"

	"github.com/globalsign/mgo/bson"
	"github.com/juliotorresmoreno/doppler/db"
	"github.com/juliotorresmoreno/doppler/model"
)

func GenerateToken(user *model.User) string {
	token := bson.NewObjectId().Hex()
	redisCli := db.GetRedisClient()
	redisCli.Set(token, user.Id, 24*time.Hour)
	return token
}
