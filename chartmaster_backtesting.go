package main

import (
	"fmt"
	"runtime"
	"time"
)

func runBacktest(
	risk, lev, accSz float64,
	userID, rid, ticker, period string,
	startTime, endTime time.Time,
	packetSize int,
	userStrat func([]Candlestick, float64, float64, float64, []float64, []float64, []float64, []float64, int, *StrategyExecutor, *interface{}, Bot, []float64, []float64, float64, float64) (map[string]map[int]string, int, map[string]string),
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

	//run strat on all candles in chunk, stream each chunk to client
	retCandles, retProfitCurve, retSimTrades, allCandles := computeBacktest(risk, lev, accSz, packetSize, userID, rid, startTime, endTime, userStrat, packetSender, &chunksArr, c, retrieveCandles)

	// Store the variables in case the user wants to store it as JSON in GCP Bucket
	saveCandlesPrepared(startTime, endTime, period, ticker, allCandles, userID)

	_, file, line, _ := runtime.Caller(0)
	go Log(fmt.Sprintf("Backtest complete %v -> %v | %v | %v | user=%v", startTime.UTC().Format(httpTimeFormat), endTime.UTC().Format(httpTimeFormat), ticker, period, userID),
		fmt.Sprintf("<%v> %v", line, file))

	sendPacketBacktest(packetSender, userID, fmt.Sprintf("%v", time.Now().UnixNano()), totalCandles, retProfitCurve[0].Data, retSimTrades[0].Data)

	// Show progress bar as finish
	progressBar(userID, rid, len(retCandles), startTime, endTime, true)

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
