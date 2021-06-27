package main

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strconv"
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

	// _, file, line, _ := runtime.Caller(0)
	// go Log(fmt.Sprint(storageObj), fmt.Sprintf("<%v> %v", line, file))

	msngr.AddToStream(botStreamName, msgs)
}

func minuteTicker(period string) *time.Ticker {

	c := make(chan time.Time, 1)
	t := &time.Ticker{C: c}
	n := time.Now().UTC()
	count := -1.0
	c <- n
	go func() {
		for {
			n := time.Now().UTC()
			if n.Second() == 0 || count == -1.0 {
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

func executeLiveStrategy(
	bot Bot,
	userStrat func([]Candlestick, float64, float64, float64, []float64, []float64, []float64, []float64, int, *StrategyExecutor, *interface{}, Bot) map[string]map[int]string) {
	if bot.Ticker == "" || bot.Period == "" {
		_, file, line, _ := runtime.Caller(0)
		go Log(loggingInJSON(fmt.Sprintf("ticker or period nil| ticker = %v, period = %v\n", bot.Ticker, bot.Period)),
			fmt.Sprintf("<%v> %v", line, file))
	}

	var fetchedCandles []Candlestick

	createJSONFile(bot.Name, bot.Period)
	timeNow := time.Now().UTC()

	//find time interval to trigger fetches
	checkCandle := fetchCandleData(bot.Ticker, bot.Period, timeNow.Add(-1*periodDurationMap[bot.Period]), timeNow.Add(-1*periodDurationMap[bot.Period])) //this fetch just to check interval
	if len(checkCandle) <= 0 {
		_, file, line, _ := runtime.Caller(0)
		go Log(loggingInJSON(fmt.Sprintf("checkCandle fetch err, exiting live strategy loop | %v %v %v", bot.Name, bot.Ticker, bot.Period)),
			fmt.Sprintf("<%v> %v", line, file))
		//TODO: update status of bot to inactive
		return
	}
	layout := "2006-01-02T15:04:05.000Z"
	str := strings.Replace(checkCandle[len(checkCandle)-1].PeriodEnd, "0000", "", 1)
	t, _ := time.Parse(layout, str) //CoinAPI's standardized time interval
	runningIndex := 0
	// _, file, line, _ := runtime.Caller(0)
	// go Log(loggingInJSON(fmt.Sprintf("Calculated time: %v", t)),
	// 	fmt.Sprintf("<%v> %v", line, file))

	for {
		//wait for current time to equal closest standardized interval time, t (only once)
		if t == time.Now().UTC() {
			//fetch closed latest candle (same as the one checked before) and previous candles to compute pivots
			data := fetchCandleData(bot.Ticker, bot.Period, t.Add(-periodDurationMap[bot.Period]*50), t.Add(-periodDurationMap[bot.Period]*1))

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

				findPivots(opens, highs, lows, closes, runningIndex, &phs, &pls, nil)

				newStratStore := PivotsStore{
					BotID:      bot.KEY,
					PivotHighs: phs,
					PivotLows:  pls,
					Opens:      opens,
					Highs:      highs,
					Lows:       lows,
					Closes:     closes,
				}
				stratStore = newStratStore

				// fmt.Printf(colorGreen+"<%v> %v %v\n"+colorReset, i, len(stratStore.(PivotsStore).PivotHighs), len(stratStore.(PivotsStore).PivotLows))
			}

			//fetch candle and run live strat on every interval tick
			for n := range minuteTicker(bot.Period).C {
				_, file, line, _ := runtime.Caller(0)
				go Log(loggingInJSON(fmt.Sprintf("[%v] Running live strat for Bot %v | %v | %v", n.UTC().Format(httpTimeFormat), bot.KEY, bot.Ticker, bot.Period)),
					fmt.Sprintf("<%v> %v", line, file))

				//check bot ID
				if bot.KEY == "" {
					_, file, line, _ := runtime.Caller(0)
					go Log(loggingInJSON("bot.KEY empty string err"),
						fmt.Sprintf("<%v> %v", line, file))
				}

				//check for shutdown cmd
				readArgs := make(map[string]string)
				readArgs["streamName"] = bot.KEY
				readArgs["start"] = "0"
				readArgs["count"] = "999999"

				var msgs []redis.XStream
				doExit := false
				_, ret, _ := msngr.ReadStream(readArgs, "strat-svc live strat loop initial candle fetch")
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

				//can continue with live strategy execute

				runningIndex++
				//fetch storage obj
				for i := len(msgs[0].Messages) - 1; i >= 0; i-- {
					if msgs[0].Messages[i].Values["StorageObj"] != nil {
						stratStore = msgs[0].Messages[i].Values["StorageObj"]
					}
				}

				//fetch latest candle to run strategy on
				fetchedCandles = fetchCandleData(bot.Ticker, bot.Period, n.Add(-periodDurationMap[bot.Period]*1), n.Add(-periodDurationMap[bot.Period]*1))

				if len(fetchedCandles) <= 0 || fetchedCandles == nil {
					_, file, line, _ := runtime.Caller(0)
					go Log(loggingInJSON(fmt.Sprintf("[%v] No candles returned", n.UTC().Format(httpTimeFormat))),
						fmt.Sprintf("<%v> %v", line, file))
					continue
				}

				//build data
				opens = append(opens, fetchedCandles[0].Open)
				highs = append(highs, fetchedCandles[0].High)
				lows = append(lows, fetchedCandles[0].Low)
				closes = append(closes, fetchedCandles[0].Close)

				stratExec := StrategyExecutor{}
				stratExec.Init(0, true)
				risk, _ := strconv.ParseFloat(bot.AccountRiskPercPerTrade, 32)
				accSz, _ := strconv.ParseFloat(bot.AccountSizePercToTrade, 32)
				leverage, _ := strconv.ParseFloat(bot.Leverage, 32)
				userStrat(fetchedCandles, risk, leverage, accSz,
					opens,
					highs,
					lows,
					closes,
					runningIndex, &stratExec, &stratStore, bot)

				//save state in strat store obj
				var readStore PivotsStore
				if ps, ok := stratStore.(PivotsStore); ok {
					readStore = ps
				} else {
					if str, okStr := stratStore.(string); okStr {
						err := json.Unmarshal([]byte(str), &readStore)
						if err != nil {
							_, file, line, _ := runtime.Caller(0)
							go Log(loggingInJSON(fmt.Sprintf("%v", err)),
								fmt.Sprintf("<%v> %v", line, file))
							continue
						}
					} else {
						_, file, line, _ := runtime.Caller(0)
						go Log(loggingInJSON(fmt.Sprintf("[%v] Cannot cast strategy storage obj", n.UTC().Format(httpTimeFormat))),
							fmt.Sprintf("<%v> %v", line, file))
						continue
					}
				}
				// fmt.Printf(colorGreen+"<%v> %v"+colorReset, runningIndex, readStore)

				newStratStore := PivotsStore{
					PivotHighs: readStore.PivotHighs,
					PivotLows:  readStore.PivotLows,
					Opens:      opens,
					Highs:      highs,
					Lows:       lows,
					Closes:     closes,
				}
				stratStore = newStratStore

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
