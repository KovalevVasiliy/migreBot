package main

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"os"
)

var ctx = context.Background()
var redisDialogueState *redis.Client = nil
var redisTmpHeadAcheEntities *redis.Client = nil

func initRedis() {
	redisPort := os.Getenv("REDIS_PORT")
	redisHost := os.Getenv("REDIS_HOST")
	Addr := fmt.Sprintf("%s:%s",redisHost, redisPort)

	redisDialogueState = redis.NewClient(&redis.Options{
		Addr:     Addr,
		Password: "",
		DB:       0,
	})

	redisTmpHeadAcheEntities = redis.NewClient(&redis.Options{
		Addr:     Addr,
		Password: "",
		DB:       1,
	})

}

func getRedisAndContext() (*redis.Client, *redis.Client, context.Context) {
	if redisDialogueState == nil ||  redisTmpHeadAcheEntities == nil {
		initRedis()
	}
	return redisDialogueState, redisTmpHeadAcheEntities, ctx
}