package main

import (
	"fmt"
	"math"
	"time"
)

func runBacktest(
	risk, lev, accSz float64,
	userID, rid, ticker, period string,
	startTime, endTime time.Time,
	packetSize int,
	userStrat func([]Candlestick, float64, float64, float64, []float64, []float64, []float64, []float64, int, *StrategyExecutor, *interface{}) map[string]map[int]string,
	packetSender func(string, string, []CandlestickChartData, []ProfitCurveData, []SimulatedTradeData),
) ([]CandlestickChartData, []ProfitCurveData, []SimulatedTradeData) {
	var allCandleData []Candlestick
	var chunksArr []*[]Candlestick

	//fetch all candle data concurrently
	concFetchCandleData(startTime, endTime, period, ticker, packetSize, &chunksArr)

	//wait for all candle data fetch complete before running strategy
	for {
		allChunksFilled := true
		for _, e := range chunksArr {
			if len(*e) <= 0 {
				allChunksFilled = false
				break
			}
		}
		if allChunksFilled {
			break
		}
	}

	for _, e := range chunksArr {
		allCandleData = append(allCandleData, *e...)
		// progressBar(userID, rid, len(allCandleData), startTime, endTime)
	}

	//run strat on all candles in chunk, stream each chunk to client
	retCandles, retProfitCurve, retSimTrades := computeBacktest(allCandleData, risk, lev, accSz, packetSize, userID, rid, startTime, endTime, userStrat, packetSender)

	fmt.Println(colorGreen + "\n!!! Backtest complete!" + colorReset)
	return retCandles, retProfitCurve, retSimTrades
}

func runScan(
	userID, rid, ticker, period string,
	startTime, endTime time.Time,
	packetSize int,
	scannerFunc func([]Candlestick, []float64, []float64, []float64, []float64, int, *interface{}) (map[string]map[int]string, PivotTrendScanDataPoint),
	packetSender func(string, string, []CandlestickChartData, []PivotTrendScanDataPoint),
) ([]CandlestickChartData, []PivotTrendScanDataPoint) {
	var allCandleData []Candlestick
	var chunksArr []*[]Candlestick

	//fetch all candle data concurrently
	concFetchCandleData(startTime, endTime, period, ticker, packetSize, &chunksArr)

	//wait for all candle data fetch complete before running strategy
	for {
		allChunksFilled := true
		for _, e := range chunksArr {
			if len(*e) <= 0 {
				allChunksFilled = false
				break
			}
		}
		if allChunksFilled {
			break
		}
	}

	for _, e := range chunksArr {
		allCandleData = append(allCandleData, *e...)
		// progressBar(userID, rid, len(allCandleData), startTime, endTime)
	}

	//run strat on all candles in chunk, stream each chunk to client
	retCandles, retScanRes := computeScan(allCandleData, packetSize, userID, rid, startTime, endTime, scannerFunc, packetSender)

	fmt.Println(colorGreen + "\n!!! Backtest complete!" + colorReset)
	return retCandles, retScanRes
}

func runBacktestSequential(
	risk, lev, accSz float64,
	userStrat func(Candlestick, float64, float64, float64, []float64, []float64, []float64, []float64, int, *StrategyExecutor, *interface{}) map[string]map[int]string,
	userID, rid, ticker, period string,
	startTime, endTime time.Time,
	packetSize int, packetSender func(string, string, []CandlestickChartData, []ProfitCurveData, []SimulatedTradeData),
) ([]CandlestickChartData, []ProfitCurveData, []SimulatedTradeData) {

	//init
	var store interface{} //save state between strategy executions on each candle
	var retCandles []CandlestickChartData
	var retProfitCurve []ProfitCurveData
	var retSimTrades []SimulatedTradeData
	retProfitCurve = []ProfitCurveData{
		{
			Label: "strat1", //TODO: prep for dynamic strategy param values
		},
	}
	retSimTrades = []SimulatedTradeData{
		{
			Label: "strat1",
		},
	}
	strategySim := StrategyExecutor{}
	strategySim.Init(accSz, false)

	//run backtest in chunks for client stream responsiveness
	allOpens := []float64{}
	allHighs := []float64{}
	allLows := []float64{}
	allCloses := []float64{}
	relIndex := 0
	lastPacketEndIndexCandles := 0
	lastPacketEndIndexPC := 0
	lastPacketEndIndexSimT := 0
	fetchCandlesStart := startTime
	for {
		if fetchCandlesStart.After(endTime) {
			break
		}

		//get all candles of chunk
		var periodCandles []Candlestick

		fetchCandlesEnd := fetchCandlesStart.Add(periodDurationMap[period] * time.Duration(packetSize))
		if fetchCandlesEnd.After(endTime) {
			fetchCandlesEnd = endTime
		}

		//check if candles exist in cache
		redisKeyPrefix := ticker + ":" + period + ":"
		testKey := redisKeyPrefix + fetchCandlesStart.Format(httpTimeFormat) + ".0000000Z"
		testRes, _ := rdbChartmaster.HGetAll(ctx, testKey).Result()
		if (testRes["open"] == "") && (testRes["close"] == "") {
			//if no data in cache, do fresh GET and save to cache
			periodCandles = fetchCandleData(ticker, period, fetchCandlesStart, fetchCandlesEnd)
		} else {
			//otherwise, get data in cache
			periodCandles = getCachedCandleData(ticker, period, fetchCandlesStart, fetchCandlesEnd)
		}

		//run strat for all chunk's candles
		var labels map[string]map[int]string
		for _, candle := range periodCandles {
			allOpens = append(allOpens, candle.Open)
			allHighs = append(allHighs, candle.High)
			allLows = append(allLows, candle.Low)
			allCloses = append(allCloses, candle.Close)
			//TODO: build results and run for different param sets
			labels = userStrat(candle, risk, lev, accSz, allOpens, allHighs, allLows, allCloses, relIndex, &strategySim, &store)

			//build display data using strategySim
			var pcData ProfitCurveDataPoint
			var simTradeData SimulatedTradeDataPoint
			retCandles, pcData, simTradeData = saveDisplayData(retCandles, &(retProfitCurve[0].Data), candle, strategySim, relIndex, labels)
			if pcData.Equity > 0 {
				retProfitCurve[0].Data = append(retProfitCurve[0].Data, pcData)
			}
			if simTradeData.DateTime != "" {
				retSimTrades[0].Data = append(retSimTrades[0].Data, simTradeData)
			}

			//absolute index from absolute start of computation period
			relIndex++
		}

		// progressBar(userID, rid, retCandles, startTime, endTime)

		//stream data back to client in every chunk
		//rm duplicates
		var uniquePCPoints []ProfitCurveDataPoint
		for i, p := range retProfitCurve[0].Data {
			if len(uniquePCPoints) == 0 {
				if i != 0 {
					uniquePCPoints = append(uniquePCPoints, p)
				}
			} else {
				var found ProfitCurveDataPoint
				for _, search := range uniquePCPoints {
					if search.Equity == p.Equity {
						found = search
					}
				}

				if found.Equity == 0 && found.DateTime == "" {
					uniquePCPoints = append(uniquePCPoints, p)
				}
			}
		}
		retProfitCurve[0].Data = uniquePCPoints

		var uniqueStPoints []SimulatedTradeDataPoint
		for i, p := range retSimTrades[0].Data {
			if len(uniqueStPoints) == 0 {
				if i != 0 {
					uniqueStPoints = append(uniqueStPoints, p)
				}
			} else {
				var found SimulatedTradeDataPoint
				for _, search := range uniqueStPoints {
					if search.DateTime == p.DateTime {
						found = search
					}
				}

				if found.EntryPrice == 0 && found.DateTime == "" {
					uniqueStPoints = append(uniqueStPoints, p)
				}
			}
		}
		retSimTrades[0].Data = uniqueStPoints

		packetEndIndex := lastPacketEndIndexCandles + packetSize
		if packetEndIndex > len(retCandles) {
			packetEndIndex = len(retCandles)
		}
		// fmt.Printf("Sending candles %v to %v\n", lastPacketEndIndexCandles, packetEndIndex)
		pcFetchEndIndex := len(retProfitCurve[0].Data)
		packetPC := retProfitCurve[0].Data[lastPacketEndIndexPC:pcFetchEndIndex]
		stFetchEndIndex := len(retSimTrades[0].Data)
		packetSt := retSimTrades[0].Data[lastPacketEndIndexSimT:stFetchEndIndex]
		packetSender(userID, rid,
			retCandles[lastPacketEndIndexCandles:packetEndIndex],
			[]ProfitCurveData{
				{
					Label: "strat1", //TODO: prep for dynamic strategy param values
					Data:  packetPC,
				},
			},
			[]SimulatedTradeData{
				{
					Label: "strat1",
					Data:  packetSt,
				},
			})

		//save last index for streaming next chunk
		lastPacketEndIndexCandles = packetEndIndex
		lastPacketEndIndexPC = int(math.Max(float64(pcFetchEndIndex-1), float64(0)))
		lastPacketEndIndexSimT = int(math.Max(float64(stFetchEndIndex-1), float64(0)))
		//increment
		fetchCandlesStart = fetchCandlesEnd.Add(periodDurationMap[period])
	}

	fmt.Println(colorGreen + "Backtest complete!" + colorReset)
	return retCandles, retProfitCurve, retSimTrades
}
