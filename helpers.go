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
	"gitlab.com/myikaco/msngr"
)

func initRedis() {
	// msngr redis streams
	if redisHostMsngr == "" {
		redisHostMsngr = "127.0.0.1"
	}
	if redisPortMsngr == "" {
		redisPortMsngr = "6379"
	}
	fmt.Println("api-gateway connecting to Redis on " + redisHostMsngr + ":" + redisPortMsngr + " - " + redisPassMsngr)
	rdbMsngr = redis.NewClient(&redis.Options{
		Addr:        redisHostMsngr + ":" + redisPortMsngr,
		Password:    redisPassMsngr,
		IdleTimeout: -1,
	})

	ctx := context.Background()
	rdbMsngr.Do(ctx, "AUTH", redisPassMsngr)
	rdbMsngr.Do(ctx, "CLIENT", "SET", "TIMEOUT", "999999999999")
	rdbMsngr.Do(ctx, "CLIENT", "SETNAME", msngr.GenerateNewConsumerID("strategy-svc"))

	// chartmaster
	if redisHostChartmaster == "" {
		redisHostMsngr = "127.0.0.1"
	}
	if redisPortChartmaster == "" {
		redisPortMsngr = "6379"
	}
	fmt.Println("api-gateway connecting to Redis on " + redisHostChartmaster + ":" + redisPortChartmaster + " - " + redisPassChartmaster)
	rdbChartmaster = redis.NewClient(&redis.Options{
		Addr:        redisHostChartmaster + ":" + redisPortChartmaster,
		Password:    redisPassChartmaster,
		IdleTimeout: -1,
	})

	rdbChartmaster.Do(ctx, "AUTH", redisPassChartmaster)
	rdbChartmaster.Do(ctx, "CLIENT", "SET", "TIMEOUT", "999999999999")
	rdbChartmaster.Do(ctx, "CLIENT", "SETNAME", msngr.GenerateNewConsumerID("strategy-svc"))
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

var nums = []rune("1234567890abcdefghijklmnopqrstuvwxyz")

func generateRandomID(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = nums[rand.Intn(len(nums))]
	}
	return string(b)
}
