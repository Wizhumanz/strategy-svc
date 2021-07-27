package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"

	"os"

	"cloud.google.com/go/datastore"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"gitlab.com/myikaco/msngr"
)

var chunkSize = 100

var redisHostMsngr = os.Getenv("REDISHOST_MSNGR")
var redisPortMsngr = os.Getenv("REDISPORT_MSNGR")
var redisPassMsngr = os.Getenv("REDISPASS_MSNGR")
var redisHostChartmaster = os.Getenv("REDISHOST_CM")
var redisPortChartmaster = os.Getenv("REDISPORT_CM")
var redisPassChartmaster = os.Getenv("REDISPASS_CM")

var newCmdStream string
var svcConsumerGroupName string
var lastIDSaveKey string
var redisConsumerID string
var minIdleAutoclaim string
var stopListenCmdChecker func([]redis.XStream, error) bool

var rdbChartmaster *redis.Client
var dsClient *datastore.Client
var ctx context.Context

var periodDurationMap = map[string]time.Duration{}
var httpTimeFormat string

var botStreamCmdHandlers []msngr.CommandHandler
var wsConnections map[string]*websocket.Conn
var wsConnectionsChartmaster map[string]*websocket.Conn
var googleProjectID = "myika-anastasia"

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}
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
	httpTimeFormat = "2006-01-02T15:04:05"

	initRedis()
	go initDatastore()

	wsConnections = make(map[string]*websocket.Conn)
	wsConnectionsChartmaster = make(map[string]*websocket.Conn)

	msngr.GoogleProjectID = "myika-anastasia"
	msngr.LoggerFunc = func(log string) {
		_, file, line, _ := runtime.Caller(0)
		_, file2, line2, _ := runtime.Caller(1)
		_, file3, line3, _ := runtime.Caller(2)
		_, file4, line4, _ := runtime.Caller(3)
		go Log(log, fmt.Sprintf("<%v> %v \n| <%v> %v \n| <%v> %v \n| <%v> %v", line, file, line2, file2, line3, file3, line4, file4))
	}
	msngr.InitRedis(redisHostMsngr, redisPortMsngr, redisPassMsngr)

	periodDurationMap["1MIN"] = 1 * time.Minute
	periodDurationMap["2MIN"] = 2 * time.Minute
	periodDurationMap["3MIN"] = 3 * time.Minute
	periodDurationMap["4MIN"] = 4 * time.Minute
	periodDurationMap["5MIN"] = 5 * time.Minute
	periodDurationMap["6MIN"] = 6 * time.Minute
	periodDurationMap["10MIN"] = 10 * time.Minute
	periodDurationMap["15MIN"] = 15 * time.Minute
	periodDurationMap["20MIN"] = 20 * time.Minute
	periodDurationMap["30MIN"] = 30 * time.Minute
	periodDurationMap["1HRS"] = 1 * time.Hour
	periodDurationMap["2HRS"] = 2 * time.Hour
	periodDurationMap["3HRS"] = 3 * time.Hour
	periodDurationMap["4HRS"] = 4 * time.Hour
	periodDurationMap["6HRS"] = 6 * time.Hour
	periodDurationMap["8HRS"] = 8 * time.Hour
	periodDurationMap["12HRS"] = 12 * time.Hour
	periodDurationMap["1DAY"] = 24 * time.Hour
	periodDurationMap["2DAY"] = 48 * time.Hour

	//note: stream listening pause + continue handlers in StatusActivateHandler
	botStatusChangeHandlers := []msngr.CommandHandler{
		{
			Command: "CMD",
			HandlerMatches: []msngr.HandlerMatch{
				{
					Matcher: func(fieldVal string) bool {
						return fieldVal == "Activate"
					},
					Handler: StatusActivateHandler,
				},
			},
		},
	}

	stopListenCmdChecker = func(msgs []redis.XStream, e error) bool {
		msg := msgs[0].Messages[0]
		doContinue := true
		cmdVal := msngr.FilterMsgVals(msg, func(k, v string) bool {
			return k == svcConsumerGroupName+"_LISTENER_CMD" || k == "CMD"
		})

		if cmdVal == "PAUSE" || cmdVal == "SHUTDOWN" {
			doContinue = false
		} else {
			doContinue = true
		}

		return doContinue
	}

	fmt.Println("Running...")
	// data, _ := rdbChartmaster.Do(ctx, "keys", "BINANCEFTS_PERP_BTC_USDT:1MIN:202*").Result()
	// fmt.Println(strings.Split(data.([]interface{})[1].(string), "N:")[1])
	// fmt.Println(data.([]interface{}))
	// earliest, latest := FindMaxAndMin(data)
	// fmt.Println(earliest, latest)

	var data = [][]string{{"day", "count"}}

	file, errors := os.Create("calendarData.csv")
	checkError("Cannot create file", errors)
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, value := range data {
		err := writer.Write(value)
		checkError("Cannot write to file", err)
	}

	csvDate()

	//create new redis consumer group for webhookTrades stream
	_, err := msngr.CreateNewConsumerGroup(newCmdStream, svcConsumerGroupName, "0")
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		go Log(fmt.Sprintf("%s redis consumer group create err = %v", svcConsumerGroupName, err.Error()),
			fmt.Sprintf("<%v> %v", line, file))
	}
	//create new redis consumer group ID
	//always create new ID because dead consumers' pending msgs will be autoclaimed by other instances
	redisConsumerID = msngr.GenerateNewConsumerID("strat")

	//live servicing

	//autoclaim pending messages from dead consumers in same group (instances of same svc)
	//listen on bot status change stream (waiting room)
	go msngr.AutoClaimMsgsLoop(newCmdStream, "strat-svc autoclaim main.go", svcConsumerGroupName, redisConsumerID, minIdleAutoclaim, "0-0", "1", botStatusChangeHandlers)

	//continuously listen for new trades to manage in bot status change stream
	go msngr.StreamListenLoop(newCmdStream, "strat-svc streamListenLoop main.go", ">", svcConsumerGroupName, redisConsumerID, "1", lastIDSaveKey, botStatusChangeHandlers, func(msgs []redis.XStream, e error) bool {
		//never kill this listen loop
		return true
	})

	//TODO: make autoclaim loop for specific bot streams

	//regular REST API
	router := mux.NewRouter().StrictSlash(true)
	router.Methods("GET").Path("/").HandlerFunc(indexHandler)

	router.Methods("GET", "OPTIONS").Path("/ws/{id}").HandlerFunc(wsConnectHandler)
	router.Methods("GET", "OPTIONS").Path("/ws-cm/{id}").HandlerFunc(wsChartmasterConnectHandler)

	router.Methods("POST", "OPTIONS").Path("/backtest").HandlerFunc(backtestHandler)
	router.Methods("POST", "OPTIONS").Path("/shareresult").HandlerFunc(shareResultHandler)
	router.Methods("GET", "OPTIONS").Path("/getshareresult").HandlerFunc(getShareResultHandler)
	router.Methods("GET", "OPTIONS").Path("/getallshareresults").HandlerFunc(getAllShareResultHandler)

	router.Methods("GET", "OPTIONS").Path("/getChartmasterTickers").HandlerFunc(getTickersHandler)
	router.Methods("GET", "OPTIONS").Path("/backtestHistory").HandlerFunc(getBacktestHistoryHandler)
	router.Methods("GET", "OPTIONS").Path("/backtestHistory/{id}").HandlerFunc(getBacktestResHandler)

	router.Methods("GET", "OPTIONS").Path("/savedCandlesHistory").HandlerFunc(getSavedCandlesHandler)
	router.Methods("POST", "OPTIONS").Path("/saveCandlesToJson").HandlerFunc(saveCandlesToJson)

	port := os.Getenv("PORT")
	fmt.Println("strategy-svc listening on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

func FindMaxAndMin(redisDates interface{}) (earliest time.Time, latest time.Time) {
	layout := "2006-01-02T15:04:05.0000000Z"
	date := redisDates.([]interface{})[0]
	if date != nil {
		earliest, _ = time.Parse(layout, strings.Split(date.(string), "N:")[1])
		latest = earliest
	} else {
		return
	}

	fmt.Println("kms")
	for _, rDate := range redisDates.([]interface{}) {
		if rDate != nil {
			dateTime := strings.Split(rDate.(string), "N:")[1]
			layout := "2006-01-02T15:04:05.0000000Z"
			formattedTime, _ := time.Parse(layout, dateTime)
			if formattedTime.Before(latest) {
				latest = formattedTime
			}
			if formattedTime.After(earliest) {
				earliest = formattedTime
			}
		}
	}
	return earliest, latest
}

func csvDate() {
	startDate := "2020-05-01T00:00:00.0000000Z"
	endDate := "2021-07-04T00:00:00.0000000Z"
	for {
		data, _ := rdbChartmaster.Do(ctx, "keys", "BINANCEFTS_PERP_BTC_USDT:1MIN:"+strings.Split(startDate, "T")[0]+"*").Result()
		if len(data.([]interface{})) == 1440 {
			// fmt.Println("Whole Day")
		} else if len(data.([]interface{})) != 0 {
			// fmt.Println("Incomplete")
		}

		layout := "2006-01-02T15:04:05.0000000Z"
		formattedTime, _ := time.Parse(layout, startDate)
		formattedTime = formattedTime.AddDate(0, 0, 1)
		startDate = formattedTime.Format(layout)

		if startDate == endDate {
			break
		}
	}
}

func checkError(message string, err error) {
	if err != nil {
		log.Fatal(message, err)
	}
}
