package main

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"gitlab.com/myikaco/msngr"
)

// logLiveStrategyExecution saves state of strategy execution loop to bot's dedicated stream in redis
func logLiveStrategyExecution(execTimestamp, storageObj, botStreamName string) {
	// add new trade info into stream (triggers other services)
	msgs := []string{}
	msgs = append(msgs, "Timestamp")
	msgs = append(msgs, execTimestamp)
	msgs = append(msgs, "StorageObj")
	msgs = append(msgs, storageObj)

	fmt.Println(storageObj)

	msngr.AddToStream(botStreamName, msgs)
}

func minuteTicker(period string) *time.Ticker {

	c := make(chan time.Time, 1)
	t := &time.Ticker{C: c}
	count := -1.0
	go func() {
		for {
			n := time.Now().UTC()
			if n.Second() == 0 {
				count += 1
			}
			if count >= periodDurationMap[period].Minutes() {
				c <- n
				count = 0
			}
			time.Sleep(time.Second)
		}
	}()
	return t
}

//1. store state of running strategy loops (across multiple instances)
//X a. api-gateway XADD trade stream ID to activeBots stream (waiting room)
//X b. strat-svc listen on specific bot's stream, adds msg on every iteration of live loop
// c. strat-svc instances check unacknowledged entries in newTrades stream, then XAUTOCLAIM old msgs in trade stream

//2. store state of storage obj + relIndex + OHLC for each running live strategy loop
// key:JSON in redis

//3. how to stop running live strategy loop when bot status changed to inactive
//X. before exec strat on each iteration, check for ending command in trade stream

func executeLiveStrategy(
	bot Bot, ticker, period string,
	userStrat func([]Candlestick, float64, float64, float64, []float64, []float64, []float64, []float64, int, *StrategyExecutor, *interface{}) map[string]map[int]string) {
	var fetchedCandles []Candlestick

	timeNow := time.Now().UTC()

	//find time interval to trigger fetches
	checkCandle := fetchCandleData(ticker, period, timeNow.Add(-1*periodDurationMap[period]), timeNow.Add(-1*periodDurationMap[period])) //this fetch just to check interval
	layout := "2006-01-02T15:04:05.000Z"
	str := strings.Replace(checkCandle[len(checkCandle)-1].PeriodEnd, "0000", "", 1)
	t, _ := time.Parse(layout, str) //CoinAPI's standardized time interval
	runningIndex := 0

	for {
		//wait for current time to equal closest standardized interval time, t (only once)
		if t == time.Now().UTC() {
			//fetch closed latest candle (same as the one checked before) and previous candles to compute pivots
			data := fetchCandleData(ticker, period, t.Add(-periodDurationMap[period]*30), t.Add(-periodDurationMap[period]*1))

			//compute some previous pivots to give strategy starting point
			var stratStore interface{}
			var opens, highs, lows, closes []float64
			for i, candle := range data {
				runningIndex = i
				opens = append(opens, candle.Open)
				highs = append(highs, candle.High)
				lows = append(lows, candle.Low)
				closes = append(closes, candle.Close)

				if stored, ok := stratStore.(PivotsStore); ok {
					stratStore = stored
				} else {
					stratStore = PivotsStore{
						PivotHighs: []int{},
						PivotLows:  []int{},
					}
				}
				phs := stratStore.(PivotsStore).PivotHighs
				pls := stratStore.(PivotsStore).PivotLows

				findPivots(opens, highs, lows, closes, runningIndex, &phs, &pls)

				newStratStore := PivotsStore{
					PivotHighs:            phs,
					PivotLows:             pls,
					LongEntryPrice:        stratStore.(PivotsStore).LongEntryPrice,
					LongSLPrice:           stratStore.(PivotsStore).LongSLPrice,
					LongPosSize:           stratStore.(PivotsStore).LongPosSize,
					MinSearchIndex:        stratStore.(PivotsStore).MinSearchIndex,
					EntryFirstPivotIndex:  stratStore.(PivotsStore).EntryFirstPivotIndex,
					EntrySecondPivotIndex: stratStore.(PivotsStore).EntrySecondPivotIndex,
					TPIndex:               stratStore.(PivotsStore).TPIndex,
					SLIndex:               stratStore.(PivotsStore).SLIndex,
				}
				stratStore = newStratStore

				fmt.Printf(colorGreen+"<%v> %v %v\n"+colorReset, i, len(stratStore.(PivotsStore).PivotHighs), len(stratStore.(PivotsStore).PivotLows))
			}

			//fetch candle and run live strat on every interval tick
			for n := range minuteTicker(period).C {
				_, file, line, _ := runtime.Caller(0)
				go Log(fmt.Sprintf("[%v] Running live strat for Bot %v | %v | %v", n.UTC().Format(httpTimeFormat), bot.KEY, ticker, period),
					fmt.Sprintf("<%v> %v", line, file))

				if bot.KEY == "" {
					_, file, line, _ := runtime.Caller(0)
					go Log("bot.KEY empty string err",
						fmt.Sprintf("<%v> %v", line, file))
				}

				//check for shutdown cmd
				readArgs := make(map[string]string)
				readArgs["streamName"] = bot.KEY
				readArgs["start"] = "0"
				readArgs["count"] = "999999"

				var msgs []redis.XStream
				doExit := false
				_, ret, _ := msngr.ReadStream(readArgs)
				if realRes, ok := ret.([]redis.XStream); ok {
					msgs = realRes
				}
				for i, m := range msgs[0].Messages {
					if i == len(msgs[0].Messages)-1 {
						if m.Values["CMD"] == "SHUTDOWN" {
							doExit = true
							break
						}
					}
				}

				//stop running live strat loop if shutdown cmd active
				if doExit {
					break
				}

				runningIndex++

				//TODO: fetch saved storage obj for strategy from redis (using msngr.ReadStream())
				for i := len(msgs[0].Messages) - 1; i >= 0; i-- {
					if msgs[0].Messages[i].Values["StorageObj"] != nil {
						stratStore = msgs[0].Messages[i].Values["StorageObj"]
					}
				}

				fetchedCandles = fetchCandleData(ticker, period, n.Add(-periodDurationMap[period]*1), n.Add(-periodDurationMap[period]*1))

				if fetchedCandles == nil {
					continue
				}
				//TODO: get bot's real settings to pass to strategy
				stratExec := StrategyExecutor{}
				stratExec.Init(0, true)
				userStrat(fetchedCandles, 0.0, 0.0, 0.0,
					[]float64{fetchedCandles[0].Open},
					[]float64{fetchedCandles[0].High},
					[]float64{fetchedCandles[0].Low},
					[]float64{fetchedCandles[0].Close},
					runningIndex, &stratExec, &stratStore)

				//save state to retrieve for next iteration
				obj, err := json.Marshal(stratStore)
				if err != nil {
					fmt.Printf(colorRed+"%v\n"+colorReset, err)
				}
				logLiveStrategyExecution(n.Format(httpTimeFormat), string(obj), fmt.Sprint(bot.KEY))
			}
			break
		}
	}
}
