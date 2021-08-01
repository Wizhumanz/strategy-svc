package main

import (
	"fmt"
	"runtime"
	"strconv"
	"time"
)

var createNewCSV int = 0

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

	// Create csv file
	csvData := []string{"Slope_EMA1", "Slope_EMA2", "Slope_EMA3", "Slope_EMA4", "Distance_Btwn_Emas", "Time", "DayOfWeek", "Month", "PivotLows", "MaxDuration", "SlPerc", "SlCooldown", "TpSingle"}
	csvFileName := startTime.Format("2006-01-02_15:04:05") + "~" + endTime.Format("2006-01-02_15:04:05") + "(" + period + ", " + ticker + ")"
	csvWrite(csvData, csvFileName)

	// pivotLowsNum := 5
	// maxDurationNum := 1000
	// slCooldown := 35
	tpCooldown := 0
	// slPercent := 1.5
	// tpSingle := 1.5

	for pivotLowsNum := 6; pivotLowsNum <= 6; pivotLowsNum++ {
		for maxDurationNum := range []float64{1500, 2000, 2500} {
			for slCooldown := range []int{10, 35} {
				for _, slPercent := range []float64{4.0, 5.0} {
					for _, tpSingle := range []float64{4.0, 6.0} {
						fmt.Printf("\npivotLowsNum: %v\n", pivotLowsNum)
						fmt.Printf("\nmaxDurationNum: %v\n", maxDurationNum)
						fmt.Printf("\nslCooldown: %v\n", slCooldown)
						fmt.Printf("\nslPercent: %v\n", slPercent)
						fmt.Printf("\ntpSingle: %v\n", tpSingle)

						//run strat on all candles in chunk, stream each chunk to client
						retCandles, retProfitCurve, retSimTrades, _ = computeBacktest(risk, lev, accSz, packetSize, userID, rid, startTime, endTime, userStrat, packetSender, &chunksArr, c, retrieveCandles, pivotLowsNum, maxDurationNum, slCooldown, tpCooldown, slPercent, tpSingle)
						// out, _ := json.Marshal(retSimTrades[0].Data)
						// fmt.Printf("\nretSimTrades: %v\n", string(out))

						for i, s := range retSimTrades[0].Data {
							if s.ExitDateTime != "" && s.RawProfitPerc > 0 {
								// out, _ := json.Marshal(retSimTrades[0].Data[i-1])
								// fmt.Printf("\nindividual: %v\n", string(out))
								ema1 := retSimTrades[0].Data[i-1].EMA1
								ema2 := retSimTrades[0].Data[i-1].EMA2
								ema3 := retSimTrades[0].Data[i-1].EMA3
								ema4 := retSimTrades[0].Data[i-1].EMA4
								previousCandle := retSimTrades[0].Data[i-1].PreviousCandle

								if ema1 == 0 || ema2 == 0 || ema3 == 0 || ema4 == 0 {
									continue
								}

								layout := "2006-01-02T15:04:05"
								time, _ := time.Parse(layout, retSimTrades[0].Data[i-1].EntryDateTime)

								min, max := findMinAndMax([]float64{ema1, ema2, ema3, ema4})

								if createNewCSV != 50000 {
									csvAdd := []string{fmt.Sprint(ema1 - previousCandle.EMA1), fmt.Sprint(ema2 - previousCandle.EMA2), fmt.Sprint(ema3 - previousCandle.EMA3), fmt.Sprint(ema4 - previousCandle.EMA4), fmt.Sprint(max - min), strconv.Itoa(time.Hour()*60 + time.Minute()), fmt.Sprint(int(time.Weekday())), fmt.Sprint(int(time.Month())), fmt.Sprint(pivotLowsNum), strconv.Itoa(maxDurationNum), fmt.Sprint(slPercent), strconv.Itoa(slCooldown), fmt.Sprint(tpSingle)}
									csvAppend(csvAdd)

									createNewCSV++
								} else {
									csvData := []string{"Slope_Volume1", "Slope_Volume2", "Slope_Volume3", "Slope_Volume4", "Volatility", "VolumeIndex", "Time", "DayOfWeek", "Month", "PivotLows", "MaxDuration", "SlPerc", "SlCooldown", "TpSingle"}
									csvFileName := startTime.Format("2006-01-02_15:04:05") + "~" + endTime.Format("2006-01-02_15:04:05") + "(" + period + ", " + ticker + ")"

									csvWrite(csvData, csvFileName)
									csvAdd := []string{fmt.Sprint(ema1 - previousCandle.EMA1), fmt.Sprint(ema2 - previousCandle.EMA2), fmt.Sprint(ema3 - previousCandle.EMA3), fmt.Sprint(ema4 - previousCandle.EMA4), fmt.Sprint(max - min), strconv.Itoa(time.Hour()*60 + time.Minute()), fmt.Sprint(int(time.Weekday())), fmt.Sprint(int(time.Month())), fmt.Sprint(pivotLowsNum), strconv.Itoa(maxDurationNum), fmt.Sprint(slPercent), strconv.Itoa(slCooldown), fmt.Sprint(tpSingle)}
									csvAppend(csvAdd)
									createNewCSV = 0
								}
							}
						}
					}
				}
			}
		}
	}

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

func findMinAndMax(a []float64) (min float64, max float64) {
	min = a[0]
	max = a[0]
	for _, value := range a {
		if value < min {
			min = value
		}
		if value > max {
			max = value
		}
	}
	return min, max
}
