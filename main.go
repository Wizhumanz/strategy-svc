package main

import (
	"fmt"
	"log"
	"net/http"

	"os"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"gitlab.com/myikaco/msngr"
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

var colorReset = "\033[0m"
var colorRed = "\033[31m"
var colorGreen = "\033[32m"
var colorYellow = "\033[33m"
var colorBlue = "\033[34m"
var colorPurple = "\033[35m"
var colorCyan = "\033[36m"
var colorWhite = "\033[37m"

func main() {
	newTradeCmdStream = "webhookTrades"
	svcConsumerGroupName = "strategy-svc"
	lastIDSaveKey = "STRATEGY-SVC:LAST_ID"
	minIdleAutoclaim = "300000" // 5 mins
	initRedis()

	msngr.GoogleProjectID = "myika-anastasia"
	msngr.InitRedis()

	listenLoopHandlers := []msngr.CommandHandler{
		{
			Command: "CMD",
			HandlerMatches: []msngr.HandlerMatch{
				{
					Matcher: func(fieldVal string) bool {
						return fieldVal == "ENTER"
					},
					Handler: CmdEnterHandler,
				},
				{
					Matcher: func(fieldVal string) bool {
						return fieldVal == "EXIT"
					},
					Handler: CmdExitHandler,
				},
				{
					Matcher: func(fieldVal string) bool {
						return fieldVal == "TP"
					},
					Handler: CmdTPHandler,
				},
				{
					Matcher: func(fieldVal string) bool {
						return fieldVal == "SL"
					},
					Handler: CmdSLHandler,
				},
			},
		},
	}

	//create new redis consumer group for webhookTrades stream
	_, err := msngr.CreateNewConsumerGroup(newTradeCmdStream, svcConsumerGroupName, "0")
	if err != nil {
		fmt.Printf("%s Redis consumer group - %v\n", svcConsumerGroupName, err.Error())
	}
	//create new redis consumer group ID
	//always create new ID because dead consumers' pending msgs will be autoclaimed
	redisConsumerID = msngr.GenerateNewConsumerID("strat")

	//live servicing

	//autoclaim pending messages from dead consumers in same group
	go msngr.AutoClaimMsgsLoop(newTradeCmdStream, svcConsumerGroupName, redisConsumerID, minIdleAutoclaim, "0-0", "1", listenLoopHandlers)

	//continuously listen for new trades to manage in webhookTrades stream
	go msngr.StreamListenLoop(newTradeCmdStream, ">", svcConsumerGroupName, redisConsumerID, "1", lastIDSaveKey, listenLoopHandlers)

	//regular REST API
	router := mux.NewRouter().StrictSlash(true)
	router.Methods("GET").Path("/").HandlerFunc(indexHandler)

	port := os.Getenv("PORT")
	fmt.Println("strategy-svc listening on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
