package main

import (
	"encoding/json"
	"fmt"
	"runtime"
	"time"
)

type StrategyDataPoint struct {
	EntryTime                          string      `json:"EntryTime"`
	SLPrice                            float64     `json:"SLPrice"`
	TPPrice                            float64     `json:"TPPrice"`
	StartTrailPerc                     float64     `json:"StartTrailPerc"`
	EntryTradeOpenCandle               Candlestick `json:"EntryTradeOpenCandle"`
	EntryLastPLIndex                   int         `json:"EntryLastPLIndex,string"`
	ActualEntryIndex                   int         `json:"ActualEntryIndex,string"`
	ExtentTime                         string      `json:"ExtentTime"`
	MaxExitIndex                       int         `json:"MaxExitIndex"`
	Duration                           float64     `json:"Duration"`
	Growth                             float64     `json:"Growth"`
	MaxDrawdownPerc                    float64     `json:"MaxDrawdownPerc"` //used to determine safe SL when trading
	BreakTime                          string      `json:"BreakTime"`
	BreakIndex                         int         `json:"BreakIndex"`
	FirstSecondEntryPivotPriceDiffPerc float64     `json:"FirstSecondEntryPivotPriceDiffPerc"`
	SecondThirdEntryPivotPriceDiffPerc float64     `json:"SecondThirdEntryPivotPriceDiffPerc"`
	FirstThirdEntryPivotPriceDiffPerc  float64     `json:"FirstThirdEntryPivotPriceDiffPerc"`
	TrailingStarted                    bool        `json:"TrailingStarted,string"`
	TrailingMaxPrice                   float64     `json:"TrailingMaxPrice,string"`
	TrailingMinPrice                   float64     `json:"TrailingMinPrice,string"`
}

type PivotsStore struct {
	PivotHighs []int
	PivotLows  []int
	Trades     []StrategyDataPoint

	Opens  []float64
	Highs  []float64
	Lows   []float64
	Closes []float64
	BotID  string
}

// calcEntry returns (posCap, posSize) so that SL only loses fixed percentage of account
func calcEntry(entryPrice, slPrice, accPercRisk, accSz float64, leverage int) (float64, float64) {
	rawRiskPerc := (entryPrice - slPrice) / entryPrice
	if rawRiskPerc < 0 {
		return -1, -1
	}

	accRiskedCap := (accPercRisk / 100) * accSz
	posCap := (accRiskedCap / rawRiskPerc) / float64(leverage)
	if posCap > accSz {
		posCap = accSz
	}
	posSize := posCap / entryPrice

	return posCap, posSize
}

// pivotWatchEntryCheck returns a slice of indexes where entry conditions met starting from startSearchIndex, if any, or empty slice if no entry is found matching the conditions.
func pivotWatchEntryCheck(lows []float64, pivotLowIndexes []int, entryWatchPivotCount, startSearchIndex int) []int {
	if len(pivotLowIndexes) < entryWatchPivotCount {
		return []int{}
	}

	//find index in pivot lows slice to start searching entry from
	retIndexes := []int{}
	searhStartPLSliIndex := 0
	startIndexSearchIncrement := 0
	for plsi, ci := range pivotLowIndexes {
		if ci > startSearchIndex {
			startIndexSearchIncrement++
			searhStartPLSliIndex = plsi
		}
		if startIndexSearchIncrement >= entryWatchPivotCount {
			break
		}
	}

	//check for lower lows in series, count = entryWatchPivotCount
	for i := searhStartPLSliIndex; i < len(pivotLowIndexes); i++ {
		//compare previous lows, count = entryWatchPivotCount
		entryConditionsTrue := true
		for j := 0; j < entryWatchPivotCount-1; j++ {
			// fmt.Printf(colorPurple+"i=%v / j=%v / pivotLowIndexes=%v\n"+colorReset, i, j, pivotLowIndexes)
			ciEarlier := pivotLowIndexes[i-j-1]
			ciLater := pivotLowIndexes[i-j]
			if !(lows[ciLater] < lows[ciEarlier]) {
				entryConditionsTrue = false
				break
			}
		}

		if !entryConditionsTrue {
			continue
		} else {
			retIndexes = append(retIndexes, pivotLowIndexes[i])
		}
	}

	return retIndexes
}

//return signature: (label, bars back to add label, storage obj to pass to next func call/iteration)
func strat1(
	candles []Candlestick, risk, lev, accSz float64,
	open, high, low, close []float64,
	relCandleIndex int,
	strategy *StrategyExecutor,
	storage *interface{}, bot Bot) map[string]map[int]string {
	//TODO: pass these 2 from frontend
	strategy.OrderSlippagePerc = 0.15
	strategy.ExchangeTradeFeePerc = 0.075

	maxDurationCandles := 600
	tpPerc := 0.5

	newLabels := map[string]map[int]string{
		"top":    map[int]string{},
		"middle": map[int]string{},
		"bottom": map[int]string{},
	}
	var stored PivotsStore
	//prep storage var
	switch (*storage).(type) {
	case PivotsStore:
		stored = (*storage).(PivotsStore)
	case string:
		json.Unmarshal([]byte((*storage).(string)), &stored)
	default:
		_, file, line, _ := runtime.Caller(0)
		go Log(loggingInJSON("Unknown type, go kys."), fmt.Sprintf("<%v> %v", line, file))
	}
	if len(stored.PivotHighs) == 0 {
		stored.PivotHighs = []int{}
	}
	if len(stored.PivotLows) == 0 {
		stored.PivotLows = []int{}
	}

	// newLabels["middle"][0] = fmt.Sprintf("%v", relCandleIndex)
	// //TP cooldown labels
	// if relCandleIndex <= (stored.TPIndex + tpTradeCooldownCandles) {
	// 	newLabels["middle"][0] = "й"
	// }
	// //SL cooldown labels
	// if relCandleIndex <= (stored.SLIndex + slTradeCooldownCandles) {
	// 	newLabels["middle"][0] = "ч"
	// }

	//calculate pivots
	newLabels, _ = findPivots(open, high, low, close, relCandleIndex, &(stored.PivotHighs), &(stored.PivotLows), newLabels)

	if len(stored.PivotLows) >= 3 {
		if strategy.GetPosLongSize() > 0 {
			//manage pos
			// fmt.Printf(colorYellow+"checking existing trend %v %v\n"+colorReset, relCandleIndex, candles[len(candles)-1].DateTime)
			latestEntryData := stored.Trades[len(stored.Trades)-1]

			//check sl + tp + max duration
			breakIndex, action := checkTrendBreak(latestEntryData, relCandleIndex, relCandleIndex-2, candles)
			if breakIndex > 0 {
				breakTrend(candles, breakIndex, relCandleIndex, &newLabels, &latestEntryData)
				stored.Trades = append(stored.Trades, latestEntryData)

				closePrice := -1.0
				switch action {
				case "SL":
					closePrice = latestEntryData.SLPrice
				case "TP":
					closePrice = latestEntryData.TPPrice
				case "MAX":
					//TODO: temp switch action name
					action = "SL"
					closePrice = close[relCandleIndex]
				}

				if closePrice > 0 {
					(*strategy).CloseLong(closePrice, 100, relCandleIndex, action, candles[len(candles)-1].DateTime(), bot)
				} else {
					_, file, line, _ := runtime.Caller(0)
					go Log(loggingInJSON(fmt.Sprintf("%v !!! strategy close price < 0 / entryData= %v", relCandleIndex, latestEntryData)), fmt.Sprintf("<%v> %v", line, file))
				}
			}
		} else {
			// fmt.Printf(colorCyan+"<%v> SEARCH new entry\n", relCandleIndex)
			possibleEntryIndexes := pivotWatchEntryCheck(low, stored.PivotLows, 3, 0)

			//for each eligible PL after last trade's exit index, run scan to check trend
			for _, pli := range possibleEntryIndexes {
				var lastTradeExitIndex int
				if len(stored.Trades) == 0 {
					lastTradeExitIndex = 0
				} else {
					lastTradeExitIndex = stored.Trades[len(stored.Trades)-1].ActualEntryIndex
				}
				// fmt.Printf(colorYellow+"<%v> checking pli= %v / lastEntryIndex= %v\n allPossibleEntries= %v\n"+colorReset, relCandleIndex, pli, lastTrendEntryIndex, possibleEntryIndexes)
				if pli <= lastTradeExitIndex {
					continue
				}

				newEntryData := StrategyDataPoint{}
				newEntryData = logScanEntry(relCandleIndex, pli, candles, possibleEntryIndexes, stored.Trades, &newEntryData, &newLabels, maxDurationCandles, 0.985, float64(1+(tpPerc/100)))
				newEntryData.ActualEntryIndex = relCandleIndex

				stored.Trades = append(stored.Trades, newEntryData)

				//enter long
				(*strategy).Buy(close[relCandleIndex], newEntryData.SLPrice, newEntryData.TPPrice, risk, int(lev), relCandleIndex, true, bot)
			}
		}
	}

	*storage = stored
	// if relCandleIndex < 250 && relCandleIndex > 120 {
	// 	fmt.Printf(colorRed+"<%v> pl=%v\nph=%v\n"+colorReset, relCandleIndex, stored.PivotLows, stored.PivotHighs)
	// }
	return newLabels
}

// SCANNING //

type PivotTrendScanStore struct {
	PivotHighs    []int
	PivotLows     []int
	ScanPoints    []StrategyDataPoint
	WatchingTrend bool
}

func logScanEntry(relCandleIndex, entryIndex int, candles []Candlestick, pivotLows []int, dataPoints []StrategyDataPoint, retData *StrategyDataPoint, newLabels *(map[string]map[int]string), maxDurationCandles int, slPerc, tpPerc float64) StrategyDataPoint {
	// fmt.Printf(colorGreen+"<%v> adding %+v\n"+colorReset, relCandleIndex, retData)

	duplicateFound := false
	for _, v := range dataPoints {
		if v.EntryLastPLIndex == entryIndex {
			duplicateFound = true
			break
		}
	}
	if duplicateFound {
		return StrategyDataPoint{}
	}

	//find actual entry index from pivot low
	actualEntryIndex := -1
	for i := entryIndex + 1; i < len(candles); i++ {
		if candles[i].High > candles[entryIndex].High {
			actualEntryIndex = i
			break
		}
	}

	if actualEntryIndex > 0 {
		retData.EntryTime = candles[actualEntryIndex].DateTime()
		retData.EntryTradeOpenCandle = candles[actualEntryIndex]
		retData.EntryLastPLIndex = entryIndex
		retData.ActualEntryIndex = actualEntryIndex

		if len(pivotLows) >= 3 {
			plSliEntryIndex := len(pivotLows) - 1
			for i := 0; i < len(pivotLows)-1; i++ {
				if pivotLows[len(pivotLows)-1-i] == entryIndex {
					plSliEntryIndex = len(pivotLows) - 1 - i
					break
				}
			}
			if (plSliEntryIndex - 2) > 0 {
				firstEntryPivot := candles[pivotLows[plSliEntryIndex-2]].Low
				secondEntryPivot := candles[pivotLows[plSliEntryIndex-1]].Low
				thirdEntryPivot := candles[pivotLows[plSliEntryIndex]].Low

				retData.FirstSecondEntryPivotPriceDiffPerc = ((firstEntryPivot - secondEntryPivot) / firstEntryPivot) * 100
				retData.SecondThirdEntryPivotPriceDiffPerc = ((secondEntryPivot - thirdEntryPivot) / secondEntryPivot) * 100
				retData.FirstThirdEntryPivotPriceDiffPerc = ((firstEntryPivot - thirdEntryPivot) / firstEntryPivot) * 100
			}
		}

		retData.MaxExitIndex = actualEntryIndex + maxDurationCandles

		retData.SLPrice = slPerc * candles[relCandleIndex].Close
		retData.TPPrice = tpPerc * candles[relCandleIndex].Close

		(*newLabels)["middle"][relCandleIndex-actualEntryIndex] = fmt.Sprintf(">/%v", retData.ActualEntryIndex)
	}

	// fmt.Printf(colorYellow+"<%v> retData= %+v\n"+colorReset, retData.EntryTradeOpenCandle.DateTime(), retData)
	return *retData
}

func checkTrendBreak(entryData StrategyDataPoint, relCandleIndex, startCheckIndex int, candles []Candlestick) (int, string) {
	// if relCandleIndex < 2100 && relCandleIndex > 1550 {
	// 	fmt.Printf(colorPurple+"<%v> checkSL sl= %v / startCheckIndex= %v / entryData= %+v\n", relCandleIndex, slPrice, startCheckIndex, entryData)
	// }

	//check max index
	if relCandleIndex >= entryData.MaxExitIndex && entryData.MaxExitIndex != 0 {
		return relCandleIndex, "MAX"
	}

	for i := startCheckIndex; i <= relCandleIndex; i++ {
		if candles[i].Low <= entryData.SLPrice && entryData.SLPrice > 0 {
			return i, "SL"
		}

		if candles[i].High >= entryData.TPPrice && entryData.TPPrice > 0 {
			return i, "TP"
		}
	}

	return -1, ""
}

func breakTrend(candles []Candlestick, breakIndex, relCandleIndex int, newLabels *(map[string]map[int]string), retData *StrategyDataPoint) {
	(*retData).BreakIndex = breakIndex
	(*retData).BreakTime = candles[breakIndex].DateTime()

	//find highest point between second entry pivot and trend break
	trendExtentIndex := retData.ActualEntryIndex //rolling compare of highest high index
	for i := retData.ActualEntryIndex + 1; i < breakIndex; i++ {
		if candles[i].High > candles[trendExtentIndex].High {
			trendExtentIndex = i
		}
	}
	(*newLabels)["middle"][relCandleIndex-trendExtentIndex] = fmt.Sprintf("$/%v", retData.ActualEntryIndex)
	(*retData).ExtentTime = candles[trendExtentIndex].DateTime()
	// fmt.Printf(colorRed+"actEntry=%v / extentIndex=%v\n"+colorReset, retData.ActualEntryIndex, trendExtentIndex)

	//find lowest point between entry and extent
	maxDrawdownIndex := retData.ActualEntryIndex //rolling compare of highest high index
	for i := retData.ActualEntryIndex + 1; i < trendExtentIndex; i++ {
		if candles[i].Low < candles[maxDrawdownIndex].Low {
			maxDrawdownIndex = i
		}
	}
	(*newLabels)["middle"][relCandleIndex-maxDrawdownIndex] = fmt.Sprintf("?/%v", retData.ActualEntryIndex)
	(*retData).MaxDrawdownPerc = ((candles[retData.ActualEntryIndex].Close - candles[maxDrawdownIndex].Low) / candles[retData.ActualEntryIndex].Close) * 100

	(*retData).Growth = ((candles[trendExtentIndex].High - retData.EntryTradeOpenCandle.Close) / retData.EntryTradeOpenCandle.Close) * 100
	// fmt.Printf(colorGreen+"break= %v / extent= %v / high[extent]= %v / entryClose=%v\n"+colorReset, breakIndex, trendExtentIndex, high[trendExtentIndex], retData.EntryTradeOpenCandle.Close)

	// trendEndTime, _ := time.Parse(httpTimeFormat, candles[breakIndex].DateTime())
	entryTime, _ := time.Parse(httpTimeFormat, retData.EntryTime)
	trendExtentTime, _ := time.Parse(httpTimeFormat, candles[trendExtentIndex].DateTime())
	(*retData).Duration = trendExtentTime.Sub(entryTime).Minutes() //log extent duration, not whole trade duration
	(*newLabels)["bottom"][relCandleIndex-breakIndex] = fmt.Sprintf("X/%v", retData.ActualEntryIndex)

	// fmt.Printf(colorGreen+"<%v> retData= %+v\n"+colorReset, retData.BreakTime, retData)
}

func contains(sli []int, find int) bool {
	found := false
	for _, e := range sli {
		if e == find {
			found = true
			break
		}
	}
	return found
}

func scanPivotTrends(
	candles []Candlestick,
	open, high, low, close []float64,
	relCandleIndex int,
	storage *interface{}) (map[string]map[int]string, StrategyDataPoint) {
	// exitWatchPivots := 3
	// checkTrendBreakFromStartingPivots := false
	// minEntryPivotsDiffPerc := float64(0)
	// maxEntryPivotsDiffPerc := 0.5
	maxDurationCandles := 600

	retData := StrategyDataPoint{}
	stored, ok := (*storage).(PivotTrendScanStore)
	if !ok {
		if relCandleIndex == 0 {
			stored.PivotHighs = []int{}
			stored.PivotLows = []int{}
		} else {
			fmt.Printf("storage obj assertion fail\n")
			return nil, StrategyDataPoint{}
		}
	}

	newLabels := map[string]map[int]string{

		"middle": map[int]string{},
		"bottom": map[int]string{},
	}

	newLabels, _ = findPivots(open, high, low, close, relCandleIndex, &(stored.PivotHighs), &(stored.PivotLows), newLabels)
	// newLabels["middle"][0] = fmt.Sprintf("%v", relCandleIndex)

	if len(stored.PivotLows) >= 3 {
		if stored.WatchingTrend {
			//manage/watch ongoing trend
			// fmt.Printf(colorYellow+"checking existing trend %v %v\n"+colorReset, relCandleIndex, candles[len(candles)-1].DateTime)
			retData = stored.ScanPoints[len(stored.ScanPoints)-1]

			//check sl
			breakIndex, _ := checkTrendBreak(retData, relCandleIndex, relCandleIndex-2, candles)
			if breakIndex > 0 {
				breakTrend(candles, breakIndex, relCandleIndex, &newLabels, &retData)
				//reset
				stored.WatchingTrend = false
				stored.ScanPoints[len(stored.ScanPoints)-1].BreakIndex = breakIndex
			} else {
				retData = StrategyDataPoint{}
			}
		} else {
			// fmt.Printf(colorCyan+"<%v> SEARCH new entry\n", relCandleIndex)
			possibleEntryIndexes := pivotWatchEntryCheck(low, stored.PivotLows, 3, 0)
			//for each eligible PL after last trend's entry index, run scan to check trend
			for _, pli := range possibleEntryIndexes {
				var lastTrendEntryIndex int
				if len(stored.ScanPoints) == 0 {
					lastTrendEntryIndex = 0
				} else {
					lastTrendEntryIndex = stored.ScanPoints[len(stored.ScanPoints)-1].ActualEntryIndex
				}
				// fmt.Printf(colorYellow+"<%v> checking pli= %v / lastEntryIndex= %v\n allPossibleEntries= %v\n"+colorReset, relCandleIndex, pli, lastTrendEntryIndex, possibleEntryIndexes)
				if pli < lastTrendEntryIndex {
					continue
				}

				newEntryData := logScanEntry(relCandleIndex, pli, candles, possibleEntryIndexes, stored.ScanPoints, &retData, &newLabels, maxDurationCandles, 0.995, -1)
				stored.ScanPoints = append(stored.ScanPoints, retData)
				stored.WatchingTrend = true

				breakIndex, _ := checkTrendBreak(newEntryData, relCandleIndex, newEntryData.ActualEntryIndex+1, candles)
				if breakIndex > 0 {
					breakTrend(candles, breakIndex, relCandleIndex, &newLabels, &retData)
					//reset
					stored.WatchingTrend = false
					stored.ScanPoints[len(stored.ScanPoints)-1].BreakIndex = breakIndex
				} else {
					retData = StrategyDataPoint{}
				}
			}
		}
	}

	*storage = stored
	// if len(newLabels["middle"]) > 0 {
	// fmt.Printf(colorYellow+"<%v> labels= %v\n"+colorReset, relCandleIndex, newLabels)
	// }
	// fmt.Printf(colorRed+"<%v> len(scanPoints)= %v\n"+colorReset, relCandleIndex, len(stored.ScanPoints))
	return newLabels, retData
}
