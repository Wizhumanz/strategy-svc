package main

import (
	"encoding/json"
	"fmt"
	"runtime"
	"time"
)

type PivotsStore struct {
	PivotHighs     []int
	PivotLows      []int
	LongEntryPrice float64
	LongSLPrice    float64
	LongPosSize    float64

	MinSearchIndex        int
	EntryFirstPivotIndex  int
	EntrySecondPivotIndex int
	TPIndex               int
	SLIndex               int

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
	//TODO: entry watch should look for 3 lower lows in a row

	//TODO: pass these 2 from frontend
	strategy.OrderSlippagePerc = 0.15
	strategy.ExchangeTradeFeePerc = 0.075

	minEntryPivotsDiffPerc := float64(0)
	maxEntryPivotsDiffPerc := 0.5

	tpTradeCooldownCandles := 5
	slTradeCooldownCandles := 9
	tpPerc := 0.75
	slPrevPLAbovePerc := 0.8

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

	newLabels["middle"][0] = fmt.Sprintf("%v", relCandleIndex)
	//TP cooldown labels
	if relCandleIndex <= (stored.TPIndex + tpTradeCooldownCandles) {
		newLabels["middle"][0] = "й"
	}
	//SL cooldown labels
	if relCandleIndex <= (stored.SLIndex + slTradeCooldownCandles) {
		newLabels["middle"][0] = "ч"
	}

	//calculate pivots
	newLabels, _ = findPivots(open, high, low, close, relCandleIndex, &(stored.PivotHighs), &(stored.PivotLows), newLabels)

	//TESTs
	(*strategy).Buy(close[relCandleIndex], 0.9*close[relCandleIndex], -1, risk, int(lev), relCandleIndex, true, bot)

	if len(stored.PivotLows) >= 2 {
		//manage/watch ongoing trend
		if strategy.GetPosLongSize() > 0 {
			// fmt.Printf(colorYellow+"checking existing trend %v %v\n"+colorReset, relCandleIndex, candles[len(candles)-1].DateTime)

			//check SL, TP
			tpPrice := (1 + (tpPerc / 100)) * stored.LongEntryPrice
			if low[relCandleIndex] <= stored.LongSLPrice {
				(*strategy).CloseLong(stored.LongSLPrice, 100, relCandleIndex, "SL", candles[len(candles)-1].DateTime(), bot)
				stored.MinSearchIndex = stored.EntrySecondPivotIndex
				stored.SLIndex = relCandleIndex
				stored.TPIndex = 0
				stored.EntryFirstPivotIndex = 0
				stored.EntrySecondPivotIndex = 0
				stored.LongEntryPrice = 0
				stored.LongSLPrice = 0
			} else if high[relCandleIndex] >= tpPrice {
				(*strategy).CloseLong(tpPrice, 100, relCandleIndex, "TP", candles[len(candles)-1].DateTime(), bot)
				stored.MinSearchIndex = stored.EntrySecondPivotIndex
				stored.TPIndex = relCandleIndex
				stored.SLIndex = 0
				stored.EntryFirstPivotIndex = 0
				stored.EntrySecondPivotIndex = 0
				stored.LongEntryPrice = 0
				stored.LongSLPrice = 0
			}
		} else {
			// fmt.Printf("finding new trend %v %v\n", relCandleIndex, candles[len(candles)-1].DateTime)

			//find new trend to watch
			latestPLIndex := stored.PivotLows[len(stored.PivotLows)-1]
			prevPLIndex := stored.PivotLows[len(stored.PivotLows)-2]
			latestPL := low[latestPLIndex]
			prevPL := low[prevPLIndex]
			entryPivotsDiffPerc := ((latestPL - prevPL) / prevPL) * 100
			if latestPL > prevPL && latestPLIndex > stored.MinSearchIndex && prevPLIndex > stored.MinSearchIndex && entryPivotsDiffPerc > minEntryPivotsDiffPerc && entryPivotsDiffPerc < maxEntryPivotsDiffPerc {
				//check timeouts
				if stored.TPIndex != 0 && relCandleIndex <= (stored.TPIndex+tpTradeCooldownCandles) {
					return newLabels
				}
				if stored.SLIndex != 0 && relCandleIndex <= (stored.SLIndex+slTradeCooldownCandles) {
					return newLabels
				}

				//enter long
				entryPrice := close[relCandleIndex]
				slPrice := prevPL + ((entryPrice - prevPL) * slPrevPLAbovePerc)
				if slPrice >= entryPrice {
					return newLabels
				}
				stored.LongSLPrice = slPrice
				stored.LongEntryPrice = entryPrice
				(*strategy).Buy(close[relCandleIndex], slPrice, -1, risk, int(lev), relCandleIndex, true, bot)
				// newLabels["middle"] = map[int]string{
				// 	0: fmt.Sprintf("%v|SL %v, TP %v", relCandleIndex, slPrice, ((1 + (tpPerc / 100)) * stored.LongEntryPrice)),
				// }

				stored.EntryFirstPivotIndex = prevPLIndex
				stored.EntrySecondPivotIndex = latestPLIndex
				stored.TPIndex = 0 //reset
				stored.SLIndex = 0
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

type PivotTrendScanDataPoint struct {
	EntryTime                          string      `json:"EntryTime"`
	EntryTradeOpenCandle               Candlestick `json:"EntryTradeOpenCandle"`
	EntryLastPLIndex                   int         `json:"EntryLastPLIndex,string"`
	ActualEntryIndex                   int         `json:"ActualEntryIndex,string"`
	ExtentTime                         string      `json:"ExtentTime"`
	Duration                           float64     `json:"Duration"`
	Growth                             float64     `json:"Growth"`
	MaxDrawdownPerc                    float64     `json:"MaxDrawdownPerc"` //used to determine safe SL when trading
	BreakTime                          string      `json:"BreakTime"`
	BreakIndex                         int         `json:"BreakIndex"`
	FirstSecondEntryPivotPriceDiffPerc float64     `json:"FirstSecondEntryPivotPriceDiffPerc"`
	SecondThirdEntryPivotPriceDiffPerc float64     `json:"SecondThirdEntryPivotPriceDiffPerc"`
	FirstThirdEntryPivotPriceDiffPerc  float64     `json:"FirstThirdEntryPivotPriceDiffPerc"`
}

type PivotTrendScanStore struct {
	PivotHighs    []int
	PivotLows     []int
	ScanPoints    []PivotTrendScanDataPoint
	WatchingTrend bool
}

func logScanEntry(relCandleIndex, entryIndex int, candles []Candlestick, pivotLows []int, stored *PivotTrendScanStore, retData *PivotTrendScanDataPoint, newLabels *(map[string]map[int]string)) PivotTrendScanDataPoint {
	// fmt.Printf(colorGreen+"<%v> adding %+v\n"+colorReset, relCandleIndex, retData)

	duplicateFound := false
	for _, v := range stored.ScanPoints {
		if v.EntryLastPLIndex == entryIndex {
			duplicateFound = true
			break
		}
	}
	if duplicateFound {
		return PivotTrendScanDataPoint{}
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
		stored.ScanPoints = append(stored.ScanPoints, *retData)

		entryPLIIndex := -1
		for i := 1; i < 0; i-- {
			checkPLI := entryPLIIndex - 1
			if contains(pivotLows, checkPLI) {
				entryPLIIndex = checkPLI
				break
			}
		}
		if entryPLIIndex > 2 {
			firstEntryPivot := candles[pivotLows[entryPLIIndex-2]].Low
			secondEntryPivot := candles[pivotLows[entryPLIIndex-1]].Low
			thirdEntryPivot := candles[pivotLows[entryPLIIndex]].Low

			retData.FirstSecondEntryPivotPriceDiffPerc = ((firstEntryPivot - secondEntryPivot) / firstEntryPivot) * 100
			retData.SecondThirdEntryPivotPriceDiffPerc = ((secondEntryPivot - thirdEntryPivot) / secondEntryPivot) * 100
			retData.FirstThirdEntryPivotPriceDiffPerc = ((firstEntryPivot - thirdEntryPivot) / firstEntryPivot) * 100
		}

		stored.WatchingTrend = true

		(*newLabels)["middle"][relCandleIndex-actualEntryIndex] = fmt.Sprintf("> /%v", relCandleIndex)
	}

	fmt.Printf(colorYellow+"<%v> retData= %+v\n"+colorReset, retData.EntryTradeOpenCandle.DateTime(), retData)
	return *retData
}

func checkSL(entryData PivotTrendScanDataPoint, relCandleIndex, startCheckIndex int, candles []Candlestick, slPrice float64) int {
	retBreakIndex := -1
	for i := startCheckIndex; i <= relCandleIndex; i++ {
		if candles[i].Low <= slPrice {
			retBreakIndex = i
			break
		}
	}

	return retBreakIndex
}

func breakTrend(candles []Candlestick, breakIndex, relCandleIndex int, newLabels *(map[string]map[int]string), retData *PivotTrendScanDataPoint, stored *PivotTrendScanStore) {
	(*retData).BreakIndex = breakIndex
	(*retData).BreakTime = candles[breakIndex].DateTime()

	//find lowest point between entry and break
	maxDrawdownIndex := retData.ActualEntryIndex //rolling compare of highest high index
	for i := retData.ActualEntryIndex + 1; i < breakIndex; i++ {
		if candles[i].Low < candles[maxDrawdownIndex].Low {
			maxDrawdownIndex = i
		}
	}
	(*newLabels)["middle"][relCandleIndex-maxDrawdownIndex] = "ж"
	(*retData).MaxDrawdownPerc = ((candles[retData.ActualEntryIndex].Close - candles[maxDrawdownIndex].Low) / candles[retData.ActualEntryIndex].Close) * 100

	//find highest point between second entry pivot and trend break
	trendExtentIndex := retData.ActualEntryIndex //rolling compare of highest high index
	for i := retData.ActualEntryIndex + 1; i < breakIndex; i++ {
		if candles[i].High > candles[trendExtentIndex].High {
			trendExtentIndex = i
		}
	}
	(*newLabels)["middle"][relCandleIndex-trendExtentIndex] = "$"
	(*retData).ExtentTime = candles[trendExtentIndex].DateTime()
	// fmt.Printf(colorRed+"actEntry=%v / extentIndex=%v\n"+colorReset, retData.ActualEntryIndex, trendExtentIndex)

	(*retData).Growth = ((candles[trendExtentIndex].High - retData.EntryTradeOpenCandle.Close) / retData.EntryTradeOpenCandle.Close) * 100
	// fmt.Printf(colorGreen+"break= %v / extent= %v / high[extent]= %v / entryClose=%v\n"+colorReset, breakIndex, trendExtentIndex, high[trendExtentIndex], retData.EntryTradeOpenCandle.Close)

	// trendEndTime, _ := time.Parse(httpTimeFormat, candles[breakIndex].DateTime())
	entryTime, _ := time.Parse(httpTimeFormat, retData.EntryTime)
	trendExtentTime, _ := time.Parse(httpTimeFormat, candles[trendExtentIndex].DateTime())
	(*retData).Duration = trendExtentTime.Sub(entryTime).Minutes() //log extent duration, not whole trade duration
	(*newLabels)["bottom"][relCandleIndex-breakIndex] = fmt.Sprintf("X /%v/%v", relCandleIndex, entryTime)

	//reset
	(*stored).WatchingTrend = false
	(*stored).ScanPoints[len((*stored).ScanPoints)-1].BreakIndex = breakIndex //don't enter with same PL as past trend, must be after break of past trend

	fmt.Printf(colorGreen+"<%v> retData= %+v\n"+colorReset, retData.BreakTime, retData)
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
	storage *interface{}) (map[string]map[int]string, PivotTrendScanDataPoint) {
	// exitWatchPivots := 3
	// checkTrendBreakFromStartingPivots := false
	// minEntryPivotsDiffPerc := float64(0)
	// maxEntryPivotsDiffPerc := 0.5

	retData := PivotTrendScanDataPoint{}
	stored, ok := (*storage).(PivotTrendScanStore)
	if !ok {
		if relCandleIndex == 0 {
			stored.PivotHighs = []int{}
			stored.PivotLows = []int{}
		} else {
			fmt.Printf("storage obj assertion fail\n")
			return nil, PivotTrendScanDataPoint{}
		}
	}
	// if len(stored.ScanPoints) <= 0 {
	// 	stored.ScanPoints = append(stored.ScanPoints, PivotTrendScanDataPoint{})
	// }

	newLabels := map[string]map[int]string{
		"top":    map[int]string{},
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
			breakIndex := checkSL(retData, relCandleIndex, relCandleIndex-2, candles, 0.99*low[retData.ActualEntryIndex])
			if breakIndex > 0 {
				breakTrend(candles, breakIndex, relCandleIndex, &newLabels, &retData, &stored)
			} else {
				retData = PivotTrendScanDataPoint{}
			}
		} else {
			// fmt.Printf(colorCyan+"<%v> SEARCH new entry\n", relCandleIndex)
			possibleEntryIndexes := pivotWatchEntryCheck(low, stored.PivotLows, 3, 0)
			if len(stored.ScanPoints) == 0 && len(possibleEntryIndexes) > 0 {
				newEntryData := logScanEntry(relCandleIndex, possibleEntryIndexes[len(possibleEntryIndexes)-1], candles, stored.PivotLows, &stored, &retData, &newLabels)
				breakIndex := checkSL(newEntryData, relCandleIndex, newEntryData.ActualEntryIndex+1, candles, 0.995*low[newEntryData.ActualEntryIndex])
				if breakIndex > 0 {
					breakTrend(candles, breakIndex, relCandleIndex, &newLabels, &retData, &stored)
				} else {
					retData = PivotTrendScanDataPoint{}
				}
			} else if len(possibleEntryIndexes) > 0 {
				//for each eligible PL after last trend's entry index, run scan to check trend
				for _, pli := range possibleEntryIndexes {
					lastTrendEntryIndex := stored.ScanPoints[len(stored.ScanPoints)-1].ActualEntryIndex
					// fmt.Printf(colorYellow+"<%v> checking pli= %v / lastEntryIndex= %v\n allPossibleEntries= %v\n"+colorReset, relCandleIndex, pli, lastTrendEntryIndex, possibleEntryIndexes)
					if pli < lastTrendEntryIndex {
						continue
					}

					newEntryData := logScanEntry(relCandleIndex, pli, candles, possibleEntryIndexes, &stored, &retData, &newLabels)
					breakIndex := checkSL(newEntryData, relCandleIndex, newEntryData.ActualEntryIndex+1, candles, 0.995*low[newEntryData.ActualEntryIndex])
					if breakIndex > 0 {
						breakTrend(candles, breakIndex, relCandleIndex, &newLabels, &retData, &stored)
					} else {
						retData = PivotTrendScanDataPoint{}
					}
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
