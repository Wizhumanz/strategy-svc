package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/gorilla/websocket"
	"google.golang.org/api/iterator"
)

func copyObjs(base []Candlestick, copyer func(Candlestick) CandlestickChartData) []CandlestickChartData {
	var ret []CandlestickChartData
	for _, obj := range base {
		ret = append(ret, copyer(obj))
	}
	return ret
}

func cacheCandleData(candles []Candlestick, ticker, period string) {
	fmt.Printf(colorYellow+"Saving %v candles from %v to %v\n"+colorReset, len(candles), candles[0].PeriodStart, candles[len(candles)-1].PeriodStart)
	//progress indicator
	indicatorParts := 30
	totalLen := len(candles)
	if totalLen < indicatorParts {
		indicatorParts = 1
	}
	lenPart := totalLen / indicatorParts
	for i, c := range candles {
		// fmt.Println(c)
		ctx := context.Background()
		key := ticker + ":" + period + ":" + c.PeriodStart
		_, err := rdbChartmaster.HMSet(ctx, key, "open", c.Open, "high", c.High, "low", c.Low, "close", c.Close, "volume", c.Volume, "tradesCount", c.TradesCount, "timeOpen", c.TimeOpen, "timeClose", c.TimeClose, "periodStart", c.PeriodStart, "periodEnd", c.PeriodEnd).Result()
		if err != nil {
			fmt.Printf("redis cache candlestick data err: %v\n", err)
			return
		}

		if (i > 1) && ((i % lenPart) == 0) {
			fmt.Printf("Section %v of %v complete\n", (i / lenPart), indicatorParts)
		}
	}
	fmt.Println(colorGreen + "Save json to redis complete!" + colorReset)
}

func fetchCandleData(ticker, period string, start, end time.Time) []Candlestick {
	fetchEndTime := end.Add(1 * periodDurationMap[period])
	fmt.Printf("FETCHING from %v to %v\n", start.Format(httpTimeFormat), fetchEndTime.Format(httpTimeFormat))

	//send request
	base := "https://rest.coinapi.io/v1/ohlcv/BINANCEFTS_PERP_BTC_USDT/history" //TODO: build dynamically based on ticker
	full := fmt.Sprintf("%s?period_id=%s&time_start=%s&time_end=%s",
		base,
		period,
		start.Format(httpTimeFormat),
		fetchEndTime.Format(httpTimeFormat))
	fmt.Println(full)

	req, _ := http.NewRequest("GET", full, nil)
	req.Header.Add("X-CoinAPI-Key", "4D684039-406E-451F-BB2B-6BDC123808E1")
	client := &http.Client{}
	response, err := client.Do(req)

	if err != nil {
		fmt.Printf("GET candle data err %v\n", err)
		return nil
	}

	//parse data
	body, _ := ioutil.ReadAll(response.Body)
	// fmt.Println(string(body))
	var jStruct []Candlestick
	errJson := json.Unmarshal(body, &jStruct)
	if errJson != nil {
		fmt.Printf("JSON unmarshall candle data err %v\n", errJson)
	}
	//save data to cache so don't have to fetch again
	if len(jStruct) > 0 && jStruct[0].Open != 0 {
		go cacheCandleData(jStruct, ticker, period)

		//temp save to loval file to preserve CoinAPI credits
		fileName := fmt.Sprintf("%v,%v,%v,%v|%v.json", ticker, period, start, end, time.Now().Unix())
		file, _ := json.MarshalIndent(jStruct, "", " ")
		_ = ioutil.WriteFile(fileName, file, 0644)
	} else {
		fmt.Println(body)
	}

	fmt.Println("Fresh fetch complete")
	return jStruct
}

func getCachedCandleData(ticker, period string, start, end time.Time) []Candlestick {
	// fmt.Printf("CACHE getting from %v to %v\n", start.Format(httpTimeFormat), end.Format(httpTimeFormat))

	var retCandles []Candlestick
	checkEnd := end.Add(periodDurationMap[period])

	for cTime := start; cTime.Before(checkEnd); cTime = cTime.Add(periodDurationMap[period]) {
		key := ticker + ":" + period + ":" + cTime.Format(httpTimeFormat) + ".0000000Z"
		cachedData, _ := rdbChartmaster.HGetAll(ctx, key).Result()

		//if candle not found in cache, fetch new
		if cachedData["open"] == "" {
			//find end time for fetch
			// var fetchEndTime time.Time
			calcTime := cTime
			for {
				calcTime = calcTime.Add(periodDurationMap[period])
				key := ticker + ":" + period + ":" + calcTime.Format(httpTimeFormat) + ".0000000Z" //TODO: update for diff period
				cached, _ := rdbChartmaster.HGetAll(ctx, key).Result()
				//find index where next cache starts again, or break if passed end time of backtest
				if (cached["open"] != "") || (calcTime.After(end)) {
					// fetchEndTime = calcTime
					break
				}
			}
			// //fetch missing candles
			// fetchedCandles := fetchCandleData(ticker, period, cTime, fetchEndTime)
			// retCandles = append(retCandles, fetchedCandles...)
			// //start getting cache again from last fetch time
			// cTime = fetchEndTime.Add(-periodDurationMap[period])
		} else {
			// fmt.Println("IN CACHE")
			newCandle := Candlestick{}
			newCandle.Create(cachedData)
			retCandles = append(retCandles, newCandle)
		}
	}

	// fmt.Printf("CACHE fetch DONE %v to %v\n", start.Format(httpTimeFormat), end.Format(httpTimeFormat))
	return retCandles
}

func saveDisplayData(cArr []CandlestickChartData, profitCurve *[]ProfitCurveDataPoint, c Candlestick, strat StrategyExecutor, relIndex int, labels map[string]map[int]string) ([]CandlestickChartData, ProfitCurveDataPoint, SimulatedTradeDataPoint) {
	//candlestick
	retCandlesArr := cArr
	newCandleD := CandlestickChartData{
		DateTime: c.DateTime,
		Open:     c.Open,
		High:     c.High,
		Low:      c.Low,
		Close:    c.Close,
	}
	//strategy enter/exit
	if strat.Actions[relIndex].Action == "ENTER" {
		newCandleD.StratEnterPrice = strat.Actions[relIndex].Price
	}
	if strat.Actions[relIndex].Action == "SL" || strat.Actions[relIndex].Action == "TP" {
		newCandleD.StratExitPrice = strat.Actions[relIndex].Price
	}
	retCandlesArr = append(retCandlesArr, newCandleD)
	//candle label
	if len(retCandlesArr) > 0 {
		if len(labels["top"]) > 0 {
			labelBB := 0
			labelText := ""
			for bb, txt := range labels["top"] {
				labelBB = bb
				labelText = txt
			}

			index := len(retCandlesArr) - labelBB - 1
			// fmt.Printf("TOP labelBB = %v\n", len(retCandlesArr), labelBB)
			if index >= 0 {
				if retCandlesArr[index].LabelTop != "" {
					retCandlesArr[index].LabelTop = retCandlesArr[index].LabelTop + "-" + labelText
				} else {
					retCandlesArr[index].LabelTop = labelText
				}
			}
		}

		if len(labels["middle"]) > 0 {
			labelBB := 0
			labelText := ""
			for bb, txt := range labels["middle"] {
				labelBB = bb
				labelText = txt
			}

			index := len(retCandlesArr) - labelBB - 1
			// fmt.Printf("MID labelBB = %v\n", len(retCandlesArr), labelBB)
			if index >= 0 {
				if retCandlesArr[index].LabelMiddle != "" {
					retCandlesArr[index].LabelMiddle = retCandlesArr[index].LabelMiddle + "-" + labelText
				} else {
					retCandlesArr[index].LabelMiddle = labelText
				}
			}
		}

		if len(labels["bottom"]) > 0 {
			labelBB := 0
			labelText := ""
			for bb, txt := range labels["bottom"] {
				labelBB = bb
				labelText = txt
			}

			index := len(retCandlesArr) - labelBB - 1
			// fmt.Printf("BOTTOM labelBB = %v\n", len(retCandlesArr), labelBB)
			if index >= 0 {
				if retCandlesArr[index].LabelBottom != "" {
					retCandlesArr[index].LabelBottom = retCandlesArr[index].LabelBottom + "-" + labelText
				} else {
					retCandlesArr[index].LabelBottom = labelText
				}
			}
		}
	}
	// fmt.Printf("A: %v", strat.GetEquity())

	//profit curve
	var pd ProfitCurveDataPoint
	if profitCurve != nil {
		//only add data point if changed from last point OR 1st or 2nd datapoint
		if (strat.GetTotalEquity() != 0) && (len(*profitCurve) == 0) && (relIndex != 0) {
			pd = ProfitCurveDataPoint{
				DateTime: c.DateTime,
				Equity:   strat.GetTotalEquity(),
			}
		} else if (relIndex == 0) || (strat.GetTotalEquity() != (*profitCurve)[len(*profitCurve)-1].Equity) {
			pd = ProfitCurveDataPoint{
				DateTime: c.DateTime,
				Equity:   strat.GetTotalEquity(),
			}
		}
	}

	//sim trades
	sd := SimulatedTradeDataPoint{}
	if len(strat.Actions) > 0 {
		if strat.Actions[relIndex].Action == "SL" || strat.Actions[relIndex].Action == "TP" {
			//find entry conditions
			var entryPrice float64
			var size float64
			for i := 1; i < relIndex; i++ {
				checkAction := strat.Actions[relIndex-i]
				if checkAction.Action == "ENTER" {
					entryPrice = checkAction.Price
					size = checkAction.PosSize
					break
				}
			}

			sd.DateTime = c.DateTime
			sd.Direction = "LONG" //TODO: fix later when strategy changes
			sd.EntryPrice = entryPrice
			sd.ExitPrice = strat.Actions[relIndex].Price
			sd.PosSize = size
			sd.RiskedEquity = size * entryPrice
			sd.RawProfitPerc = ((sd.ExitPrice - sd.EntryPrice) / sd.EntryPrice) * 100
		}
	}

	return retCandlesArr, pd, sd
}

func getChunkCandleData(chunkSlice *[]Candlestick, packetSize int, ticker, period string,
	startTime, endTime, fetchCandlesStart, fetchCandlesEnd time.Time) {
	var chunkCandles []Candlestick
	var candlesNotInCache []time.Time
	var candlesInCache []time.Time
	//check if candles exist in cache

	fmt.Printf(colorRed+"Attempting to fetch candles %v to %v\n"+colorReset, fetchCandlesStart, fetchCandlesEnd)

	for i := 0; i < int(fetchCandlesEnd.Sub(fetchCandlesStart).Minutes()); i++ {
		retCandles := getCachedCandleData(ticker, period, fetchCandlesStart.Add(time.Minute*time.Duration(i)), fetchCandlesStart.Add(time.Minute*time.Duration(i)))
		if len(retCandles) == 0 {
			candlesNotInCache = append(candlesNotInCache, fetchCandlesStart.Add(time.Minute*time.Duration(i)))
		} else {
			candlesInCache = append(candlesInCache, fetchCandlesStart.Add(time.Minute*time.Duration(i)))
		}
	}
	// fmt.Printf("\ncandlesNotInCache: %v \n", candlesNotInCache)
	// fmt.Printf("\ncandlesInCache: %v \n", candlesInCache)

	for i := 0; i < len(candlesNotInCache); i += 300 {
		if len(candlesNotInCache) > i+300 {
			chunkCandles = append(chunkCandles, fetchCandleData(ticker, period, candlesNotInCache[i], candlesNotInCache[i+299])...)
		} else {
			chunkCandles = append(chunkCandles, fetchCandleData(ticker, period, candlesNotInCache[i], candlesNotInCache[len(candlesNotInCache)-1])...)
		}
	}

	if len(candlesInCache) > 0 {
		chunkCandles = append(chunkCandles, getCachedCandleData(ticker, period, candlesInCache[0], candlesInCache[len(candlesInCache)-1])...)
	}

	var tempTimeArray []string
	var sortedChunkCandles []Candlestick
	for _, v := range chunkCandles {
		tempTimeArray = append(tempTimeArray, v.PeriodStart)
	}
	sort.Strings(tempTimeArray)
	for _, time := range tempTimeArray {
		for _, candle := range chunkCandles {
			if candle.PeriodStart == time {
				sortedChunkCandles = append(sortedChunkCandles, candle)
			}
		}
	}

	// fmt.Printf("\nTotal: %v \n", sortedChunkCandles)

	if len(sortedChunkCandles) == 0 {
		fmt.Printf("chunkCandles fetch err %v\n", startTime.Format(httpTimeFormat))
		return
	}

	*chunkSlice = sortedChunkCandles
}

func concFetchCandleData(startTime, endTime time.Time, period, ticker string, packetSize int, chunksArr *[]*[]Candlestick) {
	fetchCandlesStart := startTime
	for {
		if fetchCandlesStart.After(endTime) {
			break
		}

		fetchCandlesEnd := fetchCandlesStart.Add(periodDurationMap[period] * 300)
		if fetchCandlesEnd.After(endTime) {
			fetchCandlesEnd = endTime
		}
		var chunkSlice []Candlestick

		*chunksArr = append(*chunksArr, &chunkSlice)
		go getChunkCandleData(&chunkSlice, 300, ticker, period, startTime, endTime, fetchCandlesStart, fetchCandlesEnd)

		//increment
		fetchCandlesStart = fetchCandlesEnd.Add(periodDurationMap[period])
	}
}

func computeBacktest(
	allCandleData []Candlestick,
	risk, lev, accSz float64,
	packetSize int,
	userID, rid string,
	startTime, endTime time.Time,
	userStrat func([]Candlestick, float64, float64, float64, []float64, []float64, []float64, []float64, int, *StrategyExecutor, *interface{}) map[string]map[int]string,
	packetSender func(string, string, []CandlestickChartData, []ProfitCurveData, []SimulatedTradeData),
) ([]CandlestickChartData, []ProfitCurveData, []SimulatedTradeData) {
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

	allOpens := []float64{}
	allHighs := []float64{}
	allLows := []float64{}
	allCloses := []float64{}
	allCandles := []Candlestick{}
	relIndex := 0
	stratComputeStartIndex := 0
	for {
		if stratComputeStartIndex > len(allCandleData) {
			break
		}

		stratComputeEndIndex := stratComputeStartIndex + packetSize
		if stratComputeEndIndex > len(allCandleData) {
			stratComputeEndIndex = len(allCandleData)
		}
		periodCandles := allCandleData[stratComputeStartIndex:stratComputeEndIndex]

		//run strat for all chunk's candles
		var chunkAddedCandles []CandlestickChartData //separate chunk added vars to stream new data in packet only
		var chunkAddedPCData []ProfitCurveDataPoint
		var chunkAddedSTData []SimulatedTradeDataPoint
		var labels map[string]map[int]string
		for _, candle := range periodCandles {
			allOpens = append(allOpens, candle.Open)
			allHighs = append(allHighs, candle.High)
			allLows = append(allLows, candle.Low)
			allCloses = append(allCloses, candle.Close)
			allCandles = append(allCandles, candle)
			//TODO: build results and run for different param sets
			labels = userStrat(allCandles, risk, lev, accSz, allOpens, allHighs, allLows, allCloses, relIndex, &strategySim, &store)

			//build display data using strategySim
			var pcData ProfitCurveDataPoint
			var simTradeData SimulatedTradeDataPoint
			chunkAddedCandles, pcData, simTradeData = saveDisplayData(chunkAddedCandles, &chunkAddedPCData, candle, strategySim, relIndex, labels)
			if pcData.Equity > 0 {
				chunkAddedPCData = append(chunkAddedPCData, pcData)
			}
			if simTradeData.DateTime != "" {
				chunkAddedSTData = append(chunkAddedSTData, simTradeData)
			}

			//absolute index from absolute start of computation period
			relIndex++
		}

		//update more global vars
		retCandles = append(retCandles, chunkAddedCandles...)
		(retProfitCurve)[0].Data = append((retProfitCurve)[0].Data, chunkAddedPCData...)
		(retSimTrades)[0].Data = append((retSimTrades)[0].Data, chunkAddedSTData...)

		progressBar(userID, rid, len(retCandles), startTime, endTime)

		//stream data back to client in every chunk
		if chunkAddedCandles != nil {
			packetSender(userID, rid,
				chunkAddedCandles,
				[]ProfitCurveData{
					{
						Label: "strat1", //TODO: prep for dynamic strategy param values
						Data:  chunkAddedPCData,
					},
				},
				[]SimulatedTradeData{
					{
						Label: "strat1",
						Data:  chunkAddedSTData,
					},
				})

			stratComputeStartIndex = stratComputeEndIndex
		} else {
			break
		}
	}

	return retCandles, retProfitCurve, retSimTrades
}

func computeScan(
	allCandleData []Candlestick,
	packetSize int,
	userID, rid string,
	startTime, endTime time.Time,
	scannerFunc func([]Candlestick, []float64, []float64, []float64, []float64, int, *interface{}) (map[string]map[int]string, PivotTrendScanDataPoint),
	packetSender func(string, string, []CandlestickChartData, []PivotTrendScanDataPoint),
) ([]CandlestickChartData, []PivotTrendScanDataPoint) {
	var store interface{} //save state between strategy executions on each candle
	var retCandles []CandlestickChartData
	var retScanRes []PivotTrendScanDataPoint

	allOpens := []float64{}
	allHighs := []float64{}
	allLows := []float64{}
	allCloses := []float64{}
	allCandles := []Candlestick{}
	relIndex := 0
	stratComputeStartIndex := 0
	for {
		if stratComputeStartIndex > len(allCandleData) {
			break
		}

		stratComputeEndIndex := stratComputeStartIndex + packetSize
		if stratComputeEndIndex > len(allCandleData) {
			stratComputeEndIndex = len(allCandleData)
		}
		periodCandles := allCandleData[stratComputeStartIndex:stratComputeEndIndex]

		//run strat for all chunk's candles
		var chunkAddedCandles []CandlestickChartData //separate chunk added vars to stream new data in packet only
		var chunkAddedScanData []PivotTrendScanDataPoint
		var labels map[string]map[int]string
		for _, candle := range periodCandles {
			//run scanner func
			allCandles = append(allCandles, candle)
			allOpens = append(allOpens, candle.Open)
			allHighs = append(allHighs, candle.High)
			allLows = append(allLows, candle.Low)
			allCloses = append(allCloses, candle.Close)
			var pivotScanData PivotTrendScanDataPoint
			labels, pivotScanData = scannerFunc(allCandles, allOpens, allHighs, allLows, allCloses, relIndex, &store)

			//save res data
			chunkAddedCandles, _, _ = saveDisplayData(chunkAddedCandles, nil, candle, StrategyExecutor{}, relIndex, labels)
			if pivotScanData.Growth != 0 {
				chunkAddedScanData = append(chunkAddedScanData, pivotScanData)
			}

			//absolute index from absolute start of computation period
			relIndex++
		}

		//update more global vars
		retCandles = append(retCandles, chunkAddedCandles...)
		retScanRes = append(retScanRes, chunkAddedScanData...)

		progressBar(userID, rid, len(retCandles), startTime, endTime)

		//stream data back to client in every chunk
		if chunkAddedCandles != nil {
			packetSender(userID, rid, chunkAddedCandles, chunkAddedScanData)
			stratComputeStartIndex = stratComputeEndIndex
		} else {
			break
		}
	}

	return retCandles, retScanRes
}

func streamPacket(ws *websocket.Conn, chartData []interface{}, resID string) {
	packet := WebsocketPacket{
		ResultID: resID,
		Data:     chartData,
	}
	data, err := json.Marshal(packet)
	if err != nil {
		fmt.Printf(colorRed+"streamPacket err %v\n"+colorReset, err)
	}
	ws.WriteMessage(1, data)
}

func progressBar(userID, rid string, numOfCandles int, start, end time.Time) {
	progressMap := make(map[string]float64)
	var progressData []interface{}
	progressPerc := (float64(numOfCandles) - 1) / end.Sub(start).Minutes() * 100

	progressMap["Progress"] = progressPerc
	ws := wsConnectionsChartmaster[userID]

	progressData = append(progressData, progressMap)
	streamPacket(ws, progressData, rid)
}

func containsString(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func containsInt(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func streamBacktestResData(userID, rid string, c []CandlestickChartData, pc []ProfitCurveData, st []SimulatedTradeData) {
	ws := wsConnectionsChartmaster[userID]
	if ws != nil {
		//profit curve
		if len(pc) > 0 {
			var pcStreamData []interface{}
			for _, pCurve := range pc {
				pcStreamData = append(pcStreamData, pCurve)
			}
			streamPacket(ws, pcStreamData, rid)
		}

		//sim trades
		if len(st) > 0 {
			var stStreamData []interface{}
			for _, trade := range st {
				stStreamData = append(stStreamData, trade)
			}
			streamPacket(ws, stStreamData, rid)
		}

		//candlesticks
		var pushCandles []CandlestickChartData
		for _, candle := range c {
			if candle.DateTime == "" {

			} else {
				pushCandles = append(pushCandles, candle)
			}
		}
		var cStreamData []interface{}
		for _, can := range pushCandles {
			cStreamData = append(cStreamData, can)
		}
		streamPacket(ws, cStreamData, rid)
	}
}

func streamScanResData(userID, rid string, c []CandlestickChartData, scanData []PivotTrendScanDataPoint) {
	ws := wsConnectionsChartmaster[userID]
	if ws != nil {
		//scan pivot data point
		if len(scanData) > 0 {
			fmt.Printf("%v to %v\n", scanData[0].EntryTime, scanData[len(scanData)-1].EntryTime)

			var data []interface{}
			for _, e := range scanData {
				data = append(data, e)
			}
			streamPacket(ws, data, rid)
		}

		//candlesticks
		var pushCandles []CandlestickChartData
		for _, candle := range c {
			if candle.DateTime == "" {

			} else {
				pushCandles = append(pushCandles, candle)
			}
		}
		var cStreamData []interface{}
		for _, can := range pushCandles {
			cStreamData = append(cStreamData, can)
		}
		streamPacket(ws, cStreamData, rid)
	}
}

// makeBacktestResFile creates backtest result file with passed args and returns the name of the new file.
func makeBacktestResFile(c []CandlestickChartData, p []ProfitCurveData, s []SimulatedTradeData, ticker, period, start, end string) string {
	//only save candlesticks which are modified
	saveCandles := []CandlestickChartData{}
	for i, candle := range c {
		//only save first or last candles, and candles with entry/exit/label
		candleHasLabels := false
		if len(candle.LabelTop) > 0 || len(candle.LabelMiddle) > 0 || len(candle.LabelBottom) > 0 {
			candleHasLabels = true
		}
		if ((candle.StratEnterPrice != 0) || (candle.StratExitPrice != 0) || candleHasLabels) || ((i == 0) || (i == len(c)-1)) {
			saveCandles = append(saveCandles, candle)
		}
	}

	data := BacktestResFile{
		Ticker:               ticker,
		Period:               period,
		Start:                start,
		End:                  end,
		ModifiedCandlesticks: saveCandles,
		ProfitCurve:          p, //optimize for when equity doesn't change
		SimulatedTrades:      s,
	}
	file, _ := json.MarshalIndent(data, "", " ")
	fileName := fmt.Sprintf("%v.json", time.Now().Unix())
	_ = ioutil.WriteFile(fileName, file, 0644)

	return fileName
}

func saveBacktestRes(
	c []CandlestickChartData,
	p []ProfitCurveData,
	s []SimulatedTradeData,
	rid, reqBucketname, ticker, period, start, end string) {
	resFileName := makeBacktestResFile(c, p, s, ticker, period, start, end)

	storageClient, _ := storage.NewClient(ctx)
	defer storageClient.Close()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
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
	object := rid + ".json"
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

func completeBacktestResFile(
	rawData BacktestResFile,
	userID, rid string,
	packetSize int, packetSender func(string, string, []CandlestickChartData, []ProfitCurveData, []SimulatedTradeData),
) ([]CandlestickChartData, []ProfitCurveData, []SimulatedTradeData) {
	//init
	var completeCandles []CandlestickChartData
	start, _ := time.Parse(httpTimeFormat, rawData.Start)
	end, _ := time.Parse(httpTimeFormat, rawData.End)
	fetchCandlesStart := start

	//complete in chunks
	for {
		if fetchCandlesStart.After(end) {
			break
		}

		fetchCandlesEnd := fetchCandlesStart.Add(periodDurationMap[rawData.Period] * time.Duration(packetSize))
		if fetchCandlesEnd.After(end) {
			fetchCandlesEnd = end
		}

		//fetch all standard data
		var chunkCandles []CandlestickChartData
		blankCandles := copyObjs(getCachedCandleData(rawData.Ticker, rawData.Period, fetchCandlesStart, fetchCandlesEnd),
			func(obj Candlestick) CandlestickChartData {
				chartC := CandlestickChartData{
					DateTime: obj.DateTime,
					Open:     obj.Open,
					High:     obj.High,
					Low:      obj.Low,
					Close:    obj.Close,
				}
				return chartC
			})
		//update with added info if exists in res file
		for _, candle := range blankCandles {
			var candleToAdd CandlestickChartData
			for _, rCan := range rawData.ModifiedCandlesticks {
				if rCan.DateTime == candle.DateTime {
					candleToAdd = rCan
				}
			}
			if candleToAdd.DateTime == "" || candleToAdd.Open == 0 {
				candleToAdd = candle
			}

			chunkCandles = append(chunkCandles, candleToAdd)
		}
		completeCandles = append(completeCandles, chunkCandles...)

		//stream data back to client in every chunk
		// fmt.Printf("Sending candles %v to %v\n", fetchCandlesStart, fetchCandlesEnd)
		packetSender(userID, rid, chunkCandles, rawData.ProfitCurve, rawData.SimulatedTrades)

		//increment
		fetchCandlesStart = fetchCandlesEnd.Add(periodDurationMap[rawData.Period])
	}

	return completeCandles, rawData.ProfitCurve, rawData.SimulatedTrades
}

// listBuckets lists buckets in the project.
func listBuckets() ([]string, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	var buckets []string
	it := client.Buckets(ctx, googleProjectID)
	for {
		battrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		buckets = append(buckets, battrs.Name)
	}
	return buckets, nil
}

// listFiles lists objects within specified bucket.
func listFiles(bucket string) []string {
	// bucket := "bucket-name"
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		fmt.Println(err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	var buckets []string
	it := client.Bucket(bucket).Objects(ctx, nil)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err == nil {
			buckets = append(buckets, attrs.Name)
		}
	}
	return buckets
}

func deleteFile(bucket, object string) error {
	// bucket := "bucket-name"
	// object := "object-name"
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	o := client.Bucket(bucket).Object(object)
	if err := o.Delete(ctx); err != nil {
		return fmt.Errorf("Object(%q).Delete: %v", object, err)
	}
	// fmt.Fprintf(w, "Blob %v deleted.\n", object)
	return nil
}

func saveJsonToRedis() {
	data, err := ioutil.ReadFile("./mar-apr2021.json")
	if err != nil {
		fmt.Print(err)
	}

	var jStruct []Candlestick
	json.Unmarshal(data, &jStruct)
	// go cacheCandleData(jStruct, ticker, period)
}

func renameKeys() {
	keys, _ := rdbChartmaster.Keys(ctx, "*").Result()
	var splitKeys = map[string]string{}
	for _, k := range keys {
		splitKeys[k] = "BINANCEFTS_PERP_BTC_USDT:" + strings.SplitN(k, ":", 2)[1]
	}

	// for k, v := range splitKeys {
	// 	rdb.Rename(ctx, k, v)
	// }
}

func generateRandomCandles() {
	retData := []CandlestickChartData{}
	min := 500000
	max := 900000
	minChange := -40000
	maxChange := 45000
	minWick := 1000
	maxWick := 30000
	startDate := time.Date(2021, time.January, 1, 0, 0, 0, 0, time.Now().UTC().Location())
	for i := 0; i < 250; i++ {
		var new CandlestickChartData

		//body
		if i != 0 {
			startDate = startDate.AddDate(0, 0, 1)
			new = CandlestickChartData{
				DateTime: startDate.Format(httpTimeFormat),
				Open:     retData[len(retData)-1].Close,
			}
		} else {
			new = CandlestickChartData{
				DateTime: startDate.Format(httpTimeFormat),
				Open:     float64(rand.Intn(max-min+1)+min) / 100,
			}
		}
		new.Close = new.Open + (float64(rand.Intn(maxChange-minChange+1)+minChange) / 100)

		//wick
		if new.Close > new.Open {
			new.High = new.Close + (float64(rand.Intn(maxWick-minWick+1)+minWick) / 100)
			new.Low = new.Open - (float64(rand.Intn(maxWick-minWick+1)+minWick) / 100)
		} else {
			new.High = new.Open + (float64(rand.Intn(maxWick-minWick+1)+minWick) / 100)
			new.Low = new.Close - (float64(rand.Intn(maxWick-minWick+1)+minWick) / 100)
		}

		retData = append(retData, new)
	}
}

func generateRandomProfitCurve() {
	retData := []ProfitCurveData{}
	minChange := -110
	maxChange := 150
	minPeriodChange := 0
	maxPeriodChange := 4
	for j := 0; j < 10; j++ {
		startEquity := 1000
		startDate := time.Date(2021, time.January, 1, 0, 0, 0, 0, time.Now().UTC().Location())
		retData = append(retData, ProfitCurveData{
			Label: fmt.Sprintf("Param %v", j+1),
			Data:  []ProfitCurveDataPoint{},
		})

		for i := 0; i < 40; i++ {
			rand.Seed(time.Now().UTC().UnixNano())
			var new ProfitCurveDataPoint

			//randomize equity change
			if i == 0 {
				new.Equity = float64(startEquity)
			} else {
				change := float64(rand.Intn(maxChange-minChange+1) + minChange)
				latestIndex := len(retData[j].Data) - 1
				new.Equity = math.Abs(retData[j].Data[latestIndex].Equity + change)
			}

			new.DateTime = startDate.Format("2006-01-02")

			//randomize period skip
			randSkip := (rand.Intn(maxPeriodChange-minPeriodChange+1) + minPeriodChange)
			i = i + randSkip

			startDate = startDate.AddDate(0, 0, randSkip+1)
			retData[j].Data = append(retData[j].Data, new)
		}
	}
}
