package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/storage"
	"github.com/gorilla/mux"
	"google.golang.org/api/iterator"
)

func backtestHandler(w http.ResponseWriter, r *http.Request) {
	//create result ID for websocket packets + res storage
	rid := fmt.Sprintf("%v", time.Now().UnixNano())

	setupCORS(&w, r)
	if (*r).Method == "OPTIONS" {
		return
	}

	var req ComputeRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//get backtest res
	userID := req.User
	ticker := req.Ticker
	period := req.Period
	risk := req.Risk
	leverage := req.Leverage
	size := req.Size
	reqType := req.Operation
	reqProcess := req.Process
	retrieveCandles := req.RetrieveCandles

	candlePacketSize, err := strconv.Atoi(req.CandlePacketSize)
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		go Log(err.Error(), fmt.Sprintf("<%v> %v", line, file))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	start, err2 := time.Parse(httpTimeFormat, req.TimeStart)
	if err2 != nil {
		_, file, line, _ := runtime.Caller(0)
		go Log(err2.Error(), fmt.Sprintf("<%v> %v", line, file))
		_, file, line, _ = runtime.Caller(0)
		go Log(err.Error(), fmt.Sprintf("<%v> %v", line, file))
	}
	end, err3 := time.Parse(httpTimeFormat, req.TimeEnd)
	if err3 != nil {
		_, file, line, _ := runtime.Caller(0)
		go Log(err3.Error(), fmt.Sprintf("<%v> %v", line, file))
		_, file, line, _ = runtime.Caller(0)
		go Log(err.Error(), fmt.Sprintf("<%v> %v", line, file))
	}

	//strat params
	rF, _ := strconv.ParseFloat(risk, 32)
	lF, _ := strconv.ParseFloat(leverage, 32)
	szF, _ := strconv.ParseFloat(size, 32)

	// var candles []CandlestickChartData
	var profitCurve []ProfitCurveData
	var simTrades []SimulatedTradeData
	// var scanRes []PivotTrendScanDataPoint
	if reqType == "SCAN" {
		_, _ = runScan(userID, rid, ticker, period, start, end, candlePacketSize, scanPivotTrends, streamScanResData, reqProcess, retrieveCandles)
		//TODO: save scan results like backtest results?
	} else {
		_, profitCurve, simTrades = runBacktest(rF, lF, szF, userID, rid, ticker, period, start, end, candlePacketSize, strat1, streamBacktestResData, reqProcess, retrieveCandles)

		// Get all of user's shared history json data
		shareResult := getAllShareResult(userID)

		// delete an element in history if more than 10 items
		bucketName := "res-" + userID
		bucketData := listFiles(bucketName)
		if len(bucketData) >= 10+len(shareResult) {
			var firstLoop bool = true
			var EarliestFile storage.ObjectAttrs
			for _, file := range bucketData {
				if firstLoop && !contains(shareResult, strings.Split(file.Name, ".")[0]) {
					EarliestFile = *file
					firstLoop = false
				} else if !firstLoop && file.Created.Before(EarliestFile.Created) && !contains(shareResult, strings.Split(file.Name, ".")[0]) {
					EarliestFile = *file
				}
			}
			deleteFile(bucketName, EarliestFile.Name)
		}

		//save result to bucket
		go saveSharableResult(totalCandles, profitCurve, simTrades, bucketName, ticker, period, req.TimeStart, req.TimeEnd, rF, lF, szF, rid)
	}

	// return
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(rid)
}

func getAllShareResult(userID string) []string {
	// Get all of user's shared history json data
	var shareResult []string
	query := datastore.NewQuery("ShareResult").Filter("UserID =", userID)
	t := dsClient.Run(ctx, query)
	for {
		var x ShareResult
		_, err := t.Next(&x)
		if err == iterator.Done {
			break
		}
		shareResult = append(shareResult, x.ResultFileName)
	}
	return shareResult
}

func getAllShareResultHandler(w http.ResponseWriter, r *http.Request) {
	setupCORS(&w, r)
	if (*r).Method == "OPTIONS" {
		return
	}

	if flag.Lookup("test.v") != nil {
		initDatastore()
	}
	userID := r.URL.Query()["user"][0]
	shareResult := getAllShareResult(userID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(shareResult)
}

func shareResultHandler(w http.ResponseWriter, r *http.Request) {
	setupCORS(&w, r)
	if (*r).Method == "OPTIONS" {
		return
	}

	if flag.Lookup("test.v") != nil {
		initDatastore()
	}

	uniqueURL := fmt.Sprintf("%v", time.Now().UnixNano()) + generateRandomID(20)

	var share ShareResult
	err := json.NewDecoder(r.Body).Decode(&share)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// add new row to DB
	share.ShareID = uniqueURL
	kind := "ShareResult"
	newKey := datastore.IncompleteKey(kind, nil)
	if _, err := dsClient.Put(ctx, newKey, &share); err != nil {
		log.Fatalf("Failed to delete Bot: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(share)
}

func getShareResultHandler(w http.ResponseWriter, r *http.Request) {
	setupCORS(&w, r)
	if (*r).Method == "OPTIONS" {
		return
	}

	var shareResult ShareResult

	shareID := r.URL.Query()["share"][0]
	query := datastore.NewQuery("ShareResult").Filter("ShareID =", shareID)
	t := dsClient.Run(ctx, query)
	_, error := t.Next(&shareResult)
	if error != nil {
		_, file, line, _ := runtime.Caller(0)
		go Log(error.Error(), fmt.Sprintf("<%v> %v", line, file))
	}

	// candlePacketSize, err := strconv.Atoi(r.URL.Query()["candlePacketSize"][0])
	// if err != nil {
	// 	fmt.Println(err)
	// 	w.WriteHeader(http.StatusBadRequest)
	// 	return
	// }

	// candlePacketSize := 100

	//create result ID for websocket packets + res storage
	rid := fmt.Sprintf("%v", time.Now().UnixNano())

	//get backtest hist file
	storageClient, _ := storage.NewClient(ctx)
	defer storageClient.Close()
	ctx, cancel := context.WithTimeout(ctx, time.Second*1000)
	defer cancel()
	userID := shareResult.UserID
	bucketName := "res-" + userID
	backtestResID := shareResult.ResultFileName
	objName := backtestResID + ".json"
	rc, _ := storageClient.Bucket(bucketName).Object(objName).NewReader(ctx)
	defer rc.Close()

	// Progress bar tuned to show 10 percent done
	layout := "2006-01-02T15:04:05.000Z"
	startTime, _ := time.Parse(layout, "2021-01-01T00:00:00.000Z")
	endTime, _ := time.Parse(layout, "2021-01-01T00:59:00.000Z")
	progressBar(userID, rid, 6, startTime, endTime, false)

	backtestResByteArr, _ := ioutil.ReadAll(rc)

	var rawHistory historyResFile
	var risk float64
	var lev float64
	var accSize float64
	var candleData []CandlestickChartData
	var profitData []ProfitCurveData
	var simData []SimulatedTradeData

	json.Unmarshal(backtestResByteArr, &rawHistory)
	fmt.Printf("\nLen of share: %v\n", len(rawHistory.Candlestick))

	risk = rawHistory.Risk
	lev = rawHistory.Leverage
	accSize = rawHistory.AccountSize
	json.Unmarshal([]byte(rawHistory.Candlestick), &candleData)
	json.Unmarshal([]byte(rawHistory.ProfitCurve), &profitData)
	json.Unmarshal([]byte(rawHistory.SimulatedTrades), &simData)

	// Progress bar tuned to show half done
	start, _ := time.Parse(layout, "2021-01-01T00:00:00.000Z")
	end, _ := time.Parse(layout, "2021-01-01T00:59:00.000Z")
	progressBar(userID, rid, 30, start, end, false)

	// Send history data to frontend
	streamBacktestResData(userID, rid, candleData, profitData, simData)
	ret := []string{fmt.Sprint(risk), fmt.Sprint(lev), fmt.Sprint(accSize), backtestResID}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ret)
}

func getTickersHandler(w http.ResponseWriter, r *http.Request) {
	setupCORS(&w, r)
	if (*r).Method == "OPTIONS" {
		return
	}

	data, err := ioutil.ReadFile("./json-data/symbols-binance-fut-perp.json")
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		go Log(err.Error(), fmt.Sprintf("<%v> %v", line, file))
	}

	var t []CoinAPITicker
	json.Unmarshal(data, &t)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(t)
}

func getBacktestHistoryHandler(w http.ResponseWriter, r *http.Request) {
	setupCORS(&w, r)
	if (*r).Method == "OPTIONS" {
		return
	}

	userID := r.URL.Query()["user"][0]
	bucketData := listFiles("res-" + userID)

	var listName []string
	for _, l := range bucketData {
		listName = append(listName, l.Name)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(listName)
}

func getBacktestResHandler(w http.ResponseWriter, r *http.Request) {
	setupCORS(&w, r)
	if (*r).Method == "OPTIONS" {
		return
	}

	_, err := strconv.Atoi(r.URL.Query()["candlePacketSize"][0])
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		go Log(err.Error(), fmt.Sprintf("<%v> %v", line, file))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	//create result ID for websocket packets + res storage
	rid := fmt.Sprintf("%v", time.Now().UnixNano())

	//get backtest hist file
	storageClient, _ := storage.NewClient(ctx)
	defer storageClient.Close()
	ctx, cancel := context.WithTimeout(ctx, time.Second*1000)
	defer cancel()
	userID := r.URL.Query()["user"][0]
	bucketName := "res-" + userID
	backtestResID, _ := url.QueryUnescape(mux.Vars(r)["id"])
	objName := backtestResID + ".json"
	rc, _ := storageClient.Bucket(bucketName).Object(objName).NewReader(ctx)
	defer rc.Close()

	// Progress bar tuned to show 10 percent done
	layout := "2006-01-02T15:04:05.000Z"
	startTime, _ := time.Parse(layout, "2021-01-01T00:00:00.000Z")
	endTime, _ := time.Parse(layout, "2021-01-01T00:59:00.000Z")
	progressBar(userID, rid, 6, startTime, endTime, false)

	backtestResByteArr, _ := ioutil.ReadAll(rc)

	var rawHistory historyResFile
	var risk float64
	var lev float64
	var accSize float64
	var candleData []CandlestickChartData
	var profitData []ProfitCurveData
	var simData []SimulatedTradeData

	json.Unmarshal(backtestResByteArr, &rawHistory)
	fmt.Printf("\nLen of load: %v\n", len(rawHistory.Candlestick))

	risk = rawHistory.Risk
	lev = rawHistory.Leverage
	accSize = rawHistory.AccountSize
	json.Unmarshal([]byte(rawHistory.Candlestick), &candleData)
	json.Unmarshal([]byte(rawHistory.ProfitCurve), &profitData)
	json.Unmarshal([]byte(rawHistory.SimulatedTrades), &simData)

	// Progress bar tuned to show half done
	start, _ := time.Parse(layout, "2021-01-01T00:00:00.000Z")
	end, _ := time.Parse(layout, "2021-01-01T00:59:00.000Z")
	progressBar(userID, rid, 30, start, end, false)

	// Send history data to frontend
	streamBacktestResData(userID, rid, candleData, profitData, simData)

	ret := []float64{risk, lev, accSize}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ret)
}

var startTimeSave time.Time
var endTimeSave time.Time
var periodSave string
var tickerSave string
var allCandlesSave []Candlestick
var userIDSave string

func saveCandlesToJson(w http.ResponseWriter, r *http.Request) {
	setupCORS(&w, r)
	if (*r).Method == "OPTIONS" {
		return
	}

	saveCandlesBucket(allCandlesSave, "saved_candles-"+userIDSave, tickerSave, periodSave, startTimeSave.Format("2006-01-02_15:04:05"), endTimeSave.Format("2006-01-02_15:04:05"))

	_, file, line, _ := runtime.Caller(0)
	go Log("Candles Saved As JSON In Storage", fmt.Sprintf("<%v> %v", line, file))
}

func getSavedCandlesHandler(w http.ResponseWriter, r *http.Request) {
	setupCORS(&w, r)
	if (*r).Method == "OPTIONS" {
		return
	}

	userID := r.URL.Query()["user"][0]
	bucketData := listFiles("saved_candles-" + userID)

	var listName []string
	for _, l := range bucketData {
		listName = append(listName, l.Name)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(listName)
}

func saveCandlesPrepared(startTime, endTime time.Time, period, ticker string, allCandles []Candlestick, userID string) {
	startTimeSave = startTime
	endTimeSave = endTime
	periodSave = period
	tickerSave = ticker
	allCandlesSave = allCandles
	userIDSave = userID
}

func availableCandlesInRedis(w http.ResponseWriter, r *http.Request) {
	setupCORS(&w, r)
	if (*r).Method == "OPTIONS" {
		return
	}

	ticker := r.URL.Query()["ticker"][0]
	period := r.URL.Query()["period"][0]

	// Get calendar range from datastore
	var calendar Calendar
	query := datastore.NewQuery("Calendar").Filter("Ticker =", ticker).Filter("Period =", period)
	t := dsClient.Run(ctx, query)
	_, error := t.Next(&calendar)
	if error != nil {
		_, file, line, _ := runtime.Caller(0)
		Log(error.Error(), fmt.Sprintf("<%v> %v", line, file))
	}

	// Get all missing candles from redis
	ctx := context.Background()
	key := ticker + ":" + period
	missingCandlesRedis, err := rdbChartmaster.SMembers(ctx, key).Result()
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		go Log(fmt.Sprintf("redis cache candlestick data err: %v\n", err),
			fmt.Sprintf("<%v> %v", line, file))
		return
	}

	// Remove specific time and only store year, month, and day
	var missingCandlesOnlyDate []string
	for _, m := range missingCandlesRedis {
		onlyDate := strings.Split(m, "T")[0]
		if !contains(missingCandlesOnlyDate, onlyDate) {
			missingCandlesOnlyDate = append(missingCandlesOnlyDate, strings.Split(m, "T")[0])
		}
	}

	// Create data that has all days and counts
	var calendarData []CalendarData
	for _, c := range calendar.DateRange {
		dateRange := strings.Split(c, "~")
		layout := "2006-01-02T15:04:05.0000000Z"
		startRange, _ := time.Parse(layout, dateRange[0])
		endRange, _ := time.Parse(layout, dateRange[1])

		for i := startRange; i.Before(endRange); i = i.AddDate(0, 0, 1) {
			var x CalendarData

			if contains(missingCandlesOnlyDate, i.Format("2006-01-02")) {
				x.Count = "1"
			} else {
				x.Count = "2"
			}
			x.Day = i.Format("2006-01-02")
			calendarData = append(calendarData, x)
		}
	}

	// createCSVForAvailableCandles(ticker, period, calendar.DateRange)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(calendarData)
}

func createCSVForAvailableCandles(ticker, period string, dateRange []string) {
	resFileName := candlesAvailabilityResFile(ticker, period, dateRange)
	reqBucketname := "candles_availability"
	storageClient, _ := storage.NewClient(ctx)
	defer storageClient.Close()
	ctx, cancel := context.WithTimeout(ctx, time.Second*1000)
	defer cancel()

	//if bucket doesn't exist, create new
	buckets, _ := listBuckets()
	var bucketName string
	for _, bn := range buckets {
		if bn == reqBucketname {
			bucketName = bn
		}
	}
	if bucketName == "" {
		bucket := storageClient.Bucket(reqBucketname)
		if err := bucket.Create(ctx, googleProjectID, nil); err != nil {
			fmt.Printf("Failed to create bucket: %v", err)
		}
		bucketName = reqBucketname
	}

	//create obj
	object := ticker + ":" + period + ".csv"
	// Open local file
	f, err := os.Open(resFileName)
	if err != nil {
		fmt.Printf("os.Open: %v", err)
	}
	defer f.Close()
	ctx2, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()
	// upload object with storage.Writer
	wc := storageClient.Bucket(bucketName).Object(object).NewWriter(ctx2)
	if _, err = io.Copy(wc, f); err != nil {
		fmt.Printf("io.Copy: %v", err)
	}
	if err := wc.Close(); err != nil {
		fmt.Printf("Writer.Close: %v", err)
	}

	//remove local file
	_ = os.Remove(resFileName)
}

func candlesAvailabilityResFile(ticker, period string, dateRange []string) string {
	var fileString string

	// Get all missing candles from redis
	// ctx := context.Background()
	// key := ticker + ":" + period
	// missingCandlesRedis, err := rdbChartmaster.SMembers(ctx, key).Result()
	// if err != nil {
	// 	_, file, line, _ := runtime.Caller(0)
	// 	go Log(fmt.Sprintf("redis cache candlestick data err: %v\n", err),
	// 		fmt.Sprintf("<%v> %v", line, file))
	// }
	// firstRow := []string{"day", "count"}
	// fileString = "day + count"

	// csvWrite(firstRow, ticker+":"+period+".csv")
	first := true
	for _, c := range dateRange {
		dateRange := strings.Split(c, "~")
		layout := "2006-01-02T15:04:05.0000000Z"
		startRange, _ := time.Parse(layout, dateRange[0])
		endRange, _ := time.Parse(layout, dateRange[1])

		for i := startRange; i.Before(endRange); i = i.AddDate(0, 0, 1) {
			// row := []string{i.Format("2006-01-02"), "2"}
			// fileString = append(fileString, row...)

			if first {
				fileString = i.Format("2006-01-02") + "+" + "2"
				first = false
			} else {
				fileString = fileString + "=" + i.Format("2006-01-02") + "+" + "2"
			}

			// csvAppend(row, ticker+":"+period+".csv")
		}
	}

	//save candlesticks
	file, _ := json.Marshal(fileString)
	fileName := fmt.Sprintf("%v.csv", ticker+":"+period)
	_ = ioutil.WriteFile(fileName, file, 0644)

	return fileName
}
