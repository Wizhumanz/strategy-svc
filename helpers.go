package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"cloud.google.com/go/datastore"
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
	rdb.Do(ctx, "CLIENT", "SET", "TIMEOUT", "999999999999")
	rdb.Do(ctx, "CLIENT", "SETNAME", redisConsumerID)
}

func pingLoop() {
	for {
		ctx := context.Background()
		rdb.Ping(ctx).Result()
		time.Sleep(10000 * time.Millisecond)
	}
}

func initDatastore() {
	ctx = context.Background()
	var err error
	client, err = datastore.NewClient(ctx, googleProjectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
}

func setupCORS(w *http.ResponseWriter, req *http.Request) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Content-Type", "text/html; charset=utf-8")
	//(*w).Header().Set("Access-Control-Expose-Headers", "Authorization")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, auth, Cache-Control, Pragma, Expires")
}

func generateRandomID(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = nums[rand.Intn(len(nums))]
	}
	return string(b)
}
