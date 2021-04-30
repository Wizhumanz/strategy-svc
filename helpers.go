package main

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

func initRedis() {
	// default to dev redis instance
	if redisHost == "" {
		redisHost = "127.0.0.1"
	}
	if redisPort == "" {
		redisPort = "6379"
	}
	fmt.Println("msngr connecting to Redis on " + redisHost + ":" + redisPort + " - " + redisPass)
	rdb = redis.NewClient(&redis.Options{
		Addr:        redisHost + ":" + redisPort,
		Password:    redisPass,
		IdleTimeout: -1,
	})
	ctx := context.Background()
	rdb.Do(ctx, "AUTH", redisPass)
}
