package main

import (
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
var newTradeReqStream string
var svcConsumerGroupName string
var lastIDSaveKey string
var redisConsumerID string

func main() {
	newTradeReqStream = "webhookTrades"
	svcConsumerGroupName = "strategy-svc"
	lastIDSaveKey = "STRATEGY-SVC:LAST_ID"
	msngr.GoogleProjectID = "myika-anastasia"
	msngr.InitRedis()

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
	_, err := msngr.CreateNewConsumerGroup(newTradeReqStream, svcConsumerGroupName, "0")
	if err != nil {
		fmt.Printf("%s Redis consumer group - %v", svcConsumerGroupName, err.Error())
	}
	//create new redis consumer group ID
	redisConsumerID = msngr.GenerateNewConsumerID("strat")

	//continuously listen for new trades to manage in webhookTrades stream
	lastID, _ := msngr.GetLastID(lastIDSaveKey)
	if lastID == "" {
		lastID = "0"
	}
	go streamListenLoop(newTradeReqStream, lastID, svcConsumerGroupName, redisConsumerID)

	router := mux.NewRouter().StrictSlash(true)
	router.Methods("GET").Path("/").HandlerFunc(indexHandler)

	port := os.Getenv("PORT")
	fmt.Println("strategy-svc listening on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
