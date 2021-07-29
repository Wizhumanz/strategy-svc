package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime"
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

	// cam := []string{"2020-01-01T00:00:00.0000000Z~2020-03-30T23:59:00.0000000Z", "2020-05-01T00:00:00.0000000Z~2021-06-30T23:59:00.0000000Z", "2021-11-01T00:00:00.0000000Z~2021-12-01T23:59:00.0000000Z"}

	// var asdf []string
	// for _, c := range cam {
	// 	dateRange := strings.Split(c, "~")
	// 	layout := "2006-01-02T15:04:05.0000000Z"
	// 	startRange, _ := time.Parse(layout, dateRange[0])
	// 	endRange, _ := time.Parse(layout, dateRange[1])
	// 	fmt.Printf("\nstartRange: %v\n", startRange)
	// 	fmt.Printf("\nendRange: %v\n", endRange)

	// 	start, _ := time.Parse(layout, "2020-01-30T00:00:00.0000000Z")
	// 	end, _ := time.Parse(layout, "2020-04-02T00:00:00.0000000Z")
	// 	fmt.Printf("\nstart: %v\n", start)
	// 	fmt.Printf("\nend: %v\n", end)

	// 	if startRange.After(start) && endRange.After(end) && startRange.After(end) || startRange.Before(start) && endRange.Before(end) && end.Before(start) {
	// 		// Add new range
	// 		asdf = append(asdf, start.Format("2006-01-02T15:04:05.0000000Z")+"~"+end.Format("2006-01-02T15:04:05.0000000Z"))

	// 		fmt.Println("1")
	// 	} else if startRange.After(start) && endRange.After(start) && startRange.Before(end) && endRange.After(end) {
	// 		// Change beginning range
	// 		asdf = append(asdf, start.Format("2006-01-02T15:04:05.0000000Z")+"~"+endRange.Format("2006-01-02T15:04:05.0000000Z"))

	// 		fmt.Println("2")

	// 	} else if startRange.Before(start) && endRange.After(start) && startRange.Before(end) && endRange.Before(end) {
	// 		// Change end range
	// 		asdf = append(asdf, startRange.Format("2006-01-02T15:04:05.0000000Z")+"~"+end.Format("2006-01-02T15:04:05.0000000Z"))

	// 		fmt.Println("3")

	// 	} else {
	// 		asdf = append(asdf, c)

	// 		fmt.Println("4")
	// 	}
	// }

	// fmt.Println(asdf)

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
