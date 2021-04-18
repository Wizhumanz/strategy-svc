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
var lastIDSaveKey string

func main() {
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

	//continuously listen for new trades to manage in webhookTrades stream
	l, _ := msngr.GetLastID(lastIDSaveKey)
	if l == "" {
		l = "0"
	}
	go streamListenLoop("webhookTrades", l)

	router := mux.NewRouter().StrictSlash(true)
	router.Methods("GET").Path("/").HandlerFunc(indexHandler)

	port := os.Getenv("PORT")
	fmt.Println("strategy-svc listening on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
