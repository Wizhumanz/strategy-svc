package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"os"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"gitlab.com/myikaco/msngr"
	"gitlab.com/myikaco/saga"
)

var redisHost = os.Getenv("REDISHOST")
var redisPort = os.Getenv("REDISPORT")
var redisAddr = fmt.Sprintf("%s:%s", redisHost, redisPort)
var rdb *redis.Client
var newTradeCmdStream string
var svcConsumerGroupName string
var lastIDSaveKey string
var redisConsumerID string
var minIdleAutoclaim string

func main() {
	newTradeCmdStream = "webhookTrades"
	svcConsumerGroupName = "strategy-svc"
	lastIDSaveKey = "STRATEGY-SVC:LAST_ID"
	minIdleAutoclaim = "300000" // 5 mins
	msngr.GoogleProjectID = "myika-anastasia"
	msngr.InitRedis()
	initRedis()

	//init sagas
	OpenTradeSaga = saga.Saga{
		Steps: []saga.SagaStep{
			{
				Transaction:             checkModel,
				CompensatingTransaction: cancelCheckModel,
				ListenForResponse:       false,
			},
		},
	}

	//create new redis consumer group for webhookTrades stream
	_, err := msngr.CreateNewConsumerGroup(newTradeCmdStream, svcConsumerGroupName, "0")
	if err != nil {
		fmt.Printf("%s Redis consumer group - %v", svcConsumerGroupName, err.Error())
	}
	//create new redis consumer group ID
	//always create new ID because dead consumers' pending msgs will be autoclaimed
	redisConsumerID = msngr.GenerateNewConsumerID("strat")

	//live servicing

	//autoclaim pending messages from dead consumers in same group
	go autoClaimMsgsLoop(newTradeCmdStream, svcConsumerGroupName, redisConsumerID, minIdleAutoclaim, "0-0", "1")

	//continuously listen for new trades to manage in webhookTrades stream
	var ctx = context.Background()
	lastID, _ := rdb.Get(ctx, lastIDSaveKey).Result()
	if lastID == "" {
		lastID = "0"
	}
	go streamListenLoop(newTradeCmdStream, lastID, svcConsumerGroupName, redisConsumerID, "1")

	//regular REST API
	router := mux.NewRouter().StrictSlash(true)
	router.Methods("GET").Path("/").HandlerFunc(indexHandler)

	port := os.Getenv("PORT")
	fmt.Println("strategy-svc listening on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
