package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/go-redis/redis/v8"
	"gitlab.com/myikaco/msngr"
)

func initRedis() {
	// chartmaster
	if redisHostChartmaster == "" {
		redisHostMsngr = "127.0.0.1"
	}
	if redisPortChartmaster == "" {
		redisPortMsngr = "6379"
	}
	_, file, line, _ := runtime.Caller(0)
	go Log(svcConsumerGroupName+" connecting to Redis on "+redisHostChartmaster+":"+redisPortChartmaster+" - "+redisPassChartmaster,
		fmt.Sprintf("<%v> %v", line, file))

	rdbChartmaster = redis.NewClient(&redis.Options{
		Addr:        redisHostChartmaster + ":" + redisPortChartmaster,
		Password:    redisPassChartmaster,
		IdleTimeout: -1,
	})

	ctx := context.Background()
	_, redisCMErr := rdbChartmaster.Do(ctx, "AUTH", redisPassChartmaster).Result()
	if redisCMErr != nil {
		_, file, line, _ := runtime.Caller(0)
		go Log(redisCMErr.Error(),
			fmt.Sprintf("<%v> %v", line, file))
		return
	}
	rdbChartmaster.Do(ctx, "CLIENT", "SET", "TIMEOUT", "999999999999")
	rdbChartmaster.Do(ctx, "CLIENT", "SETNAME", msngr.GenerateNewConsumerID("strategy-svc"))
}

func initDatastore() {
	ctx = context.Background()
	var err error
	dsClient, err = datastore.NewClient(ctx, googleProjectID)
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

func pauseStreamListening(streamKey, callerLog string) {
	//stop other instances from listening temporarily
	stopMsg := []string{}
	stopMsg = append(stopMsg, svcConsumerGroupName+"_LISTENER_CMD")
	stopMsg = append(stopMsg, "PAUSE")
	stopMsg = append(stopMsg, "Caller")
	stopMsg = append(stopMsg, callerLog)
	stopMsg = append(stopMsg, "Timestamp")
	stopMsg = append(stopMsg, time.Now().UTC().Format(httpTimeFormat))
	msngr.AddToStream(streamKey, stopMsg)
}

func continueStreamListening(streamKey string) {
	go msngr.StreamListenLoop(streamKey, "strat-svc continueStreamListening", ">", svcConsumerGroupName, redisConsumerID, "1", "0", botStreamCmdHandlers, stopListenCmdChecker)
}
