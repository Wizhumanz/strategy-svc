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

var googleProjectID = "myika-anastasia"
var redisHost = os.Getenv("REDISHOST")
var redisPort = os.Getenv("REDISPORT")
var redisAddr = fmt.Sprintf("%s:%s", redisHost, redisPort)
var rdb *redis.Client

func main() {
	msngr.InitRedis()

	//init sagas
	OpenTradeSaga = saga.Saga{
		Steps: []saga.SagaStep{
			{Transaction: checkModel, CompensatingTransaction: cancelCheckModel},
			{Transaction: submitEntryOrder, CompensatingTransaction: cancelSubmitEntryOrder},
			{Transaction: submitExitOrder, CompensatingTransaction: cancelSubmitExitOrder},
		},
	}
	// go OpenLongSaga.Execute("1:order:1")

	router := mux.NewRouter().StrictSlash(true)
	router.Methods("GET").Path("/").HandlerFunc(indexHandler)
	router.Methods("POST").Path("/tv-hook").HandlerFunc(tvWebhookHandler)

	port := os.Getenv("PORT")
	fmt.Println("strategy-svc listening on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
