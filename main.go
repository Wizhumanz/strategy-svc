package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"os"

	"cloud.google.com/go/datastore"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"gitlab.com/myikaco/msngr"
)

var redisHost = os.Getenv("REDISHOST")
var redisPort = os.Getenv("REDISPORT")
var redisPass = os.Getenv("REDISPASS")
var rdb *redis.Client
var newCmdStream string
var svcConsumerGroupName string
var lastIDSaveKey string
var redisConsumerID string
var minIdleAutoclaim string

var rdbChartmaster *redis.Client
var client *datastore.Client
var ctx context.Context

var periodDurationMap = map[string]time.Duration{}
var httpTimeFormat string

var botStreamCmdHandlers []msngr.CommandHandler

var colorReset = "\033[0m"
var colorRed = "\033[31m"
var colorGreen = "\033[32m"
var colorYellow = "\033[33m"
var colorBlue = "\033[34m"
var colorPurple = "\033[35m"
var colorCyan = "\033[36m"
var colorWhite = "\033[37m"

func main() {
	newCmdStream = "activeBots"
	svcConsumerGroupName = "strategy-svc"
	lastIDSaveKey = "STRATEGY-SVC:LAST_ID"
	minIdleAutoclaim = "300000" // 5 mins
	initRedis()
	initDatastore()
	// go pingLoop()

	msngr.GoogleProjectID = "myika-anastasia"
	msngr.InitRedis()

	botStatusChangeHandlers := []msngr.CommandHandler{
		{
			Command: "Status",
			HandlerMatches: []msngr.HandlerMatch{
				{
					Matcher: func(fieldVal string) bool {
						return fieldVal == "Activate"
					},
					Handler: CmdEnterHandler,
				},
			},
		},
	}

	botStreamCmdHandlers = []msngr.CommandHandler{
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
	_, err := msngr.CreateNewConsumerGroup(newCmdStream, svcConsumerGroupName, "0")
	if err != nil {
		fmt.Printf("%s Redis consumer group - %v\n", svcConsumerGroupName, err.Error())
	}
	//create new redis consumer group ID
	//always create new ID because dead consumers' pending msgs will be autoclaimed
	redisConsumerID = msngr.GenerateNewConsumerID("strat")

	//live servicing

	//autoclaim pending messages from dead consumers in same group (instances of same svc)
	//listen on bot status change stream (waiting room)
	go msngr.AutoClaimMsgsLoop(newCmdStream, svcConsumerGroupName, redisConsumerID, minIdleAutoclaim, "0-0", "1", botStatusChangeHandlers)
	//continuously listen for new trades to manage in bot status change stream
	go msngr.StreamListenLoop(newCmdStream, ">", svcConsumerGroupName, redisConsumerID, "1", lastIDSaveKey, botStatusChangeHandlers)

	//TODO: make autoclaim loop for specific bot streams

	//regular REST API
	router := mux.NewRouter().StrictSlash(true)
	router.Methods("GET").Path("/").HandlerFunc(indexHandler)

	router.Methods("POST", "OPTIONS").Path("/backtest").HandlerFunc(backtestHandler)
	router.Methods("POST", "OPTIONS").Path("/shareresult").HandlerFunc(shareResultHandler)
	router.Methods("GET", "OPTIONS").Path("/getshareresult").HandlerFunc(getShareResultHandler)
	router.Methods("GET", "OPTIONS").Path("/getallshareresults").HandlerFunc(getAllShareResultHandler)

	router.Methods("GET", "OPTIONS").Path("/getChartmasterTickers").HandlerFunc(getTickersHandler)
	router.Methods("GET", "OPTIONS").Path("/backtestHistory").HandlerFunc(getBacktestHistoryHandler)
	router.Methods("GET", "OPTIONS").Path("/backtestHistory/{id}").HandlerFunc(getBacktestResHandler)

	port := os.Getenv("PORT")
	fmt.Println("strategy-svc listening on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
