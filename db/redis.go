package db

import "github.com/go-redis/redis"

var redisCli *redis.Client

func GetRedisClient() *redis.Client {
	if redisCli == nil {
		redisCli = redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "", // no password set
			DB:       0,  // use default DB
			PoolSize: 20,
		})
	}
	return redisCli
}
