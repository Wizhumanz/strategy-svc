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
	userStrat func([]Candlestick, float64, float64, float64, []float64, []float64, []float64, []float64, int, *StrategyExecutor, *interface{}, Bot) (map[string]map[int]string, int),
	packetSender func(string, string, []CandlestickChartData, []ProfitCurveData, []SimulatedTradeData),
) ([]CandlestickChartData, []ProfitCurveData, []SimulatedTradeData) {
	var chunksArr []*[]Candlestick

	totalCandles = nil

	// Channel to get timestamps for empty candles
	c := make(chan time.Time)

	//fetch all candle data concurrently
	concFetchCandleData(startTime, endTime, period, ticker, packetSize, &chunksArr, c)

	//run strat on all candles in chunk, stream each chunk to client
	retCandles, retProfitCurve, retSimTrades := computeBacktest(risk, lev, accSz, packetSize, userID, rid, startTime, endTime, userStrat, packetSender, &chunksArr, c)

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
) ([]CandlestickChartData, []StrategyDataPoint) {
	var chunksArr []*[]Candlestick

	totalCandles = nil

	// Channel to get timestamps for empty candles
	c := make(chan time.Time)
	//fetch all candle data concurrently
	concFetchCandleData(startTime, endTime, period, ticker, packetSize, &chunksArr, c)

	//run strat on all candles in chunk, stream each chunk to client
	retCandles, retScanRes := computeScan(packetSize, userID, rid, startTime, endTime, scannerFunc, packetSender, &chunksArr, c)

	_, file, line, _ := runtime.Caller(0)
	go Log(fmt.Sprintf(colorGreen+"\n!!! Scan complete!"+colorReset), fmt.Sprintf("<%v> %v", line, file))

	sendPacketScan(packetSender, userID, fmt.Sprintf("%v", time.Now().UnixNano()), totalCandles, retScanRes)

	// Show progress bar as finish
	progressBar(userID, rid, len(retCandles), startTime, endTime, true)
	return retCandles, retScanRes
}
