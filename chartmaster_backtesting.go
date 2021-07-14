package main

import (
	"fmt"
	"runtime"
	"strconv"
	"time"
)

func runBacktest(
	risk, lev, accSz float64,
	userID, rid, ticker, period string,
	startTime, endTime time.Time,
	packetSize int,
	userStrat func([]Candlestick, float64, float64, float64, []float64, []float64, []float64, []float64, int, *StrategyExecutor, *interface{}, Bot, int, int, int, int, float64, float64) (map[string]map[int]string, int),
	packetSender func(string, string, []CandlestickChartData, []ProfitCurveData, []SimulatedTradeData),
	processOption string,
	retrieveCandles bool,
) ([]CandlestickChartData, []ProfitCurveData, []SimulatedTradeData) {
	var chunksArr []*[]Candlestick

	totalCandles = nil

	// Channel to get timestamps for empty candles
	c := make(chan time.Time)

	if retrieveCandles {
		// Fetch from Cloud Storage
		fileName := startTime.Format("2006-01-02_15:04:05") + "~" + endTime.Format("2006-01-02_15:04:05") + "(" + period + ", " + ticker + ")" + ".json"
		retrieveJsonFromStorage(userID, fileName, &chunksArr)
	} else {
		//fetch all candle data concurrently
		concFetchCandleData(startTime, endTime, period, ticker, packetSize, &chunksArr, c, processOption)
	}

	var retCandles []CandlestickChartData
	var retProfitCurve []ProfitCurveData
	var retSimTrades []SimulatedTradeData
	// var allCandles []Candlestick
	csvData := [][]string{{"EMA1", "EMA2", "EMA3", "EMA4", "Time", "DayOfWeek", "Month", "PivotLows", "MaxDuration", "SlPerc", "SlCooldown", "TpSingle", "Profit Perc"}}

	// pivotLowsNum := 5
	// maxDurationNum := 1000
	// slCooldown := 35
	tpCooldown := 0
	// slPercent := 1.5
	// tpSingle := 1.5

	for pivotLowsNum := 3; pivotLowsNum <= 5; pivotLowsNum++ {
		fmt.Printf("\npivotLowsNum: %v\n", pivotLowsNum)
		for maxDurationNum := 800; maxDurationNum <= 1000; maxDurationNum += 100 {
			fmt.Printf("\nmaxDurationNum: %v\n", maxDurationNum)
			for slCooldown := 5; slCooldown <= 85; slCooldown += 20 {
				fmt.Printf("\nslCooldown: %v\n", slCooldown)
				for _, slPercent := range []float64{0.5, 0.7, 1.0, 1.5, 2.0} {
					fmt.Printf("\nslPercent: %v\n", slPercent)
					for _, tpSingle := range []float64{1.5, 2.5, 2.75} {
						fmt.Printf("\ntpSingle: %v\n", tpSingle)
						//run strat on all candles in chunk, stream each chunk to client
						retCandles, retProfitCurve, retSimTrades, _ = computeBacktest(risk, lev, accSz, packetSize, userID, rid, startTime, endTime, userStrat, packetSender, &chunksArr, c, retrieveCandles, pivotLowsNum, maxDurationNum, slCooldown, tpCooldown, slPercent, tpSingle)

						for i, s := range retSimTrades[0].Data {
							if s.ExitDateTime != "" && s.Profit > 0 {
								ema1 := retSimTrades[0].Data[i-1].EMA1
								ema2 := retSimTrades[0].Data[i-1].EMA2
								ema3 := retSimTrades[0].Data[i-1].EMA3
								ema4 := retSimTrades[0].Data[i-1].EMA4

								if ema1 == 0 || ema2 == 0 || ema3 == 0 || ema4 == 0 {
									continue
								}

								layout := "2006-01-02T15:04:05"
								time, _ := time.Parse(layout, retSimTrades[0].Data[i-1].EntryDateTime)
								csvData = append(csvData, []string{fmt.Sprint(ema1), fmt.Sprint(ema2), fmt.Sprint(ema3), fmt.Sprint(ema4), strconv.Itoa(time.Hour()*60 + time.Minute()), strconv.Itoa(int(time.Weekday())), strconv.Itoa(int(time.Month())), fmt.Sprint(pivotLowsNum), strconv.Itoa(maxDurationNum), fmt.Sprint(slPercent), strconv.Itoa(slCooldown), fmt.Sprint(tpSingle), fmt.Sprint(s.RawProfitPerc)})
							}
						}
					}
				}
			}
		}
	}

	csvWrite(csvData)
	// Store the variables in case the user wants to store it as JSON in GCP Bucket
	// saveCandlesPrepared(startTime, endTime, period, ticker, allCandles, userID)

	_, file, line, _ := runtime.Caller(0)
	go Log(fmt.Sprintf("Backtest complete %v -> %v | %v | %v | user=%v", startTime.UTC().Format(httpTimeFormat), endTime.UTC().Format(httpTimeFormat), ticker, period, userID),
		fmt.Sprintf("<%v> %v", line, file))

	// sendPacketBacktest(packetSender, userID, fmt.Sprintf("%v", time.Now().UnixNano()), totalCandles, retProfitCurve[0].Data, retSimTrades[0].Data)

	// Show progress bar as finish
	// progressBar(userID, rid, len(retCandles), startTime, endTime, true)

	return retCandles, retProfitCurve, retSimTrades
}

func runScan(
	userID, rid, ticker, period string,
	startTime, endTime time.Time,
	packetSize int,
	scannerFunc func([]Candlestick, []float64, []float64, []float64, []float64, int, *interface{}) (map[string]map[int]string, StrategyDataPoint),
	packetSender func(string, string, []CandlestickChartData, []StrategyDataPoint),
	processOption string,
	retrieveCandles bool,
) ([]CandlestickChartData, []StrategyDataPoint) {
	var chunksArr []*[]Candlestick

	totalCandles = nil

	// Channel to get timestamps for empty candles
	c := make(chan time.Time)

	if retrieveCandles {
		// Fetch from Cloud Storage
		fileName := startTime.Format("2006-01-02_15:04:05") + "~" + endTime.Format("2006-01-02_15:04:05") + "(" + period + ", " + ticker + ")" + ".json"
		retrieveJsonFromStorage(userID, fileName, &chunksArr)
	} else {
		//fetch all candle data concurrently
		concFetchCandleData(startTime, endTime, period, ticker, packetSize, &chunksArr, c, processOption)
	}

	//run strat on all candles in chunk, stream each chunk to client
	retCandles, retScanRes, allCandles := computeScan(packetSize, userID, rid, startTime, endTime, scannerFunc, packetSender, &chunksArr, c, retrieveCandles)

	// Store the variables in case the user wants to store it as JSON in GCP Bucket
	saveCandlesPrepared(startTime, endTime, period, ticker, allCandles, userID)

	_, file, line, _ := runtime.Caller(0)
	go Log(fmt.Sprintf(colorGreen+"\n!!! Scan complete!"+colorReset), fmt.Sprintf("<%v> %v", line, file))

	sendPacketScan(packetSender, userID, fmt.Sprintf("%v", time.Now().UnixNano()), totalCandles, retScanRes)

	// Show progress bar as finish
	progressBar(userID, rid, len(retCandles), startTime, endTime, true)
	return retCandles, retScanRes
}
