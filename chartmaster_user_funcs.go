package main

import (
	"encoding/json"
	"fmt"
	"math"
	"runtime"
	"sort"
	"strings"
	"time"
)

type MultiTPPoint struct {
	Order            int
	IsDone           bool
	Price            float64
	ClosePerc        float64
	CloseSize        float64
	TotalPointsInSet int
}

type StrategyDataPoint struct {
	EntryTime                         string         `json:"EntryTime"`
	SLPrice                           float64        `json:"SLPrice"`
	TPPrice                           float64        `json:"TPPrice"`
	MultiTPs                          []MultiTPPoint `json:"MultiTPs"`
	EntryTradeOpenCandle              Candlestick    `json:"EntryTradeOpenCandle"`
	EntryLastPLIndex                  int            `json:"EntryLastPLIndex,string"`
	ActualEntryIndex                  int            `json:"ActualEntryIndex,string"`
	ExtentTime                        string         `json:"ExtentTime"`
	MaxExitIndex                      int            `json:"MaxExitIndex"`
	Duration                          float64        `json:"Duration"`
	Growth                            float64        `json:"Growth"`
	MaxDrawdownPerc                   float64        `json:"MaxDrawdownPerc"` //used to determine safe SL when trading
	BreakTime                         string         `json:"BreakTime"`
	BreakIndex                        int            `json:"BreakIndex"`
	FirstLastEntryPivotPriceDiffPerc  float64        `json:"FirstLastEntryPivotPriceDiffPerc"`
	FirstToLastEntryPivotDuration     int            `json:"FirstToLastEntryPivotDuration"`
	AveragePriceDiffPercEntryPivots   float64        `json:"AveragePriceDiffPercEntryPivots"`
	TrailingStarted                   bool           `json:"TrailingStarted,string"`
	StartTrailPerc                    float64        `json:"StartTrailPerc"`
	TrailingPerc                      float64        `json:"TrailingPerc,string"`
	TrailingMax                       float64        `json:"TrailingMax,string"`
	TrailingMin                       float64        `json:"TrailingMin,string"`
	TrailingMaxDrawdownPercTillExtent float64        `json:"TrailingMaxDrawdownPercTillExtent,string"`
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

	accRisk := (accPercRisk / 100) * accSz
	posRisk := (rawRiskPerc)
	leveragedPosEquity := accRisk / posRisk

	// fmt.Printf(colorGreen+"accSz x lev= %v / accPercRisk= %v / accSz= %v / lev= %v / rawRisk= %v\n levPosEquity= %v\n"+colorReset, accSz*float64(leverage), accPercRisk, accSz, leverage, rawRiskPerc, leveragedPosEquity)

	if leveragedPosEquity > accSz*float64(leverage) {
		leveragedPosEquity = accSz * float64(leverage)
	}
	posSize := leveragedPosEquity / entryPrice

	// fmt.Printf(colorGreen+"FINAL levPosEquity= %v\n"+colorReset, leveragedPosEquity)

	return leveragedPosEquity, posSize
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
	storage *interface{}, bot Bot) (map[string]map[int]string, int) {
	//TODO: pass these 2 from frontend
	strategy.OrderSlippagePerc = 0.15
	strategy.ExchangeTradeFeePerc = 0.075

	//map of profit % TO account size perc to close (multi-tp)
	tpMap := map[float64]float64{
		1.0: 5,
		1.6: 15,
		2.5: 10,
		3.2: 20,
		3.8: 20,
		4.4: 20,
		5.2: 10,
	}

	//0.8% SL
	// tpMap := map[float64]float64{
	// 	1.3: 10,
	// 	1.9: 20,
	// 	2.5: 20,
	// 	3.3: 20,
	// 	3.9: 25,
	// 	5.7: 5,
	// }

	pivotLowsToEnter := 6
	maxDurationCandles := 1200
	slPerc := 1.5
	// startTrailPerc := 1.3
	// trailingPerc := 0.4
	slCooldownCandles := 35
	tpCooldownCandles := 35

	// tradeWindowStart := "09:00:00"
	tradeWindowStart := ""
	// tradeWindowEnd := "18:00:00"
	tradeWindowEnd := ""

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

	//calculate pivots
	newLabels, _ = findPivots(open, high, low, close, relCandleIndex, &(stored.PivotHighs), &(stored.PivotLows), newLabels)

	//TESTs
	// (*strategy).Buy(close[relCandleIndex], 0.9*close[relCandleIndex], -1, 0.9*close[relCandleIndex], 0.9*close[relCandleIndex], risk, int(lev), relCandleIndex, nil, candles[len(candles)-1], true, bot)

	// completedMultiTPs := (*strategy).Buy(close[relCandleIndex], newEntryData.SLPrice, newEntryData.TPPrice, newEntryData.StartTrailPerc, newEntryData.TrailingPerc, risk, int(lev), relCandleIndex, newEntryData.MultiTPs, candles[len(candles)-1], true, bot)

	// newLabels["middle"][0] = fmt.Sprintf("%v", relCandleIndex)

	// //TP cooldown labels
	// if relCandleIndex <= (stored.TPIndex + tpTradeCooldownCandles) {
	// 	newLabels["middle"][0] = "й"
	// }

	//SL cooldown labels
	// if relCandleIndex < 120 {
	// 	fmt.Printf("%+v\n", strategy.Actions)
	// }
	latestActions := []StrategyExecutorAction{}
	for k := 1; k < relCandleIndex; k++ {
		checkIndex := relCandleIndex - k
		if len(strategy.Actions[checkIndex]) > 0 {
			// if relCandleIndex < 50 {
			// 	fmt.Printf(colorYellow+"<%v> checking %+v from <%v>\n", relCandleIndex, strategy.Actions[checkIndex], checkIndex)
			// }
			latestActions = strategy.Actions[checkIndex]
			break
		}
	}
	// if relCandleIndex > 1600 && relCandleIndex < 1900 && len(stored.Trades) > 0 {
	// 	fmt.Printf(colorCyan+"<%v> latest= %+v\n"+colorReset, relCandleIndex, latestActions)
	// 	fmt.Printf(colorGreen+"%+v\n"+colorReset, stored.Trades[len(stored.Trades)-1].BreakIndex)
	// }
	if len(stored.Trades) > 0 && len(latestActions) > 0 && (latestActions[0].Action == "SL" && relCandleIndex <= (stored.Trades[len(stored.Trades)-1].BreakIndex+slCooldownCandles)) {
		newLabels["middle"][0] = "ч"
	} else if len(stored.Trades) > 0 && len(latestActions) > 0 && (latestActions[0].Action == "MULTI-TP" && (relCandleIndex <= (stored.Trades[len(stored.Trades)-1].BreakIndex + tpCooldownCandles))) {
		newLabels["middle"][0] = "ф"
	} else if len(stored.PivotLows) >= 4 {
		if strategy.GetPosLongSize() > 0 {
			//manage pos
			// fmt.Printf(colorYellow+"checking existing trend %v %v\n"+colorReset, relCandleIndex, candles[len(candles)-1].DateTime)
			latestEntryData := stored.Trades[len(stored.Trades)-1]

			//check sl + tp + max duration
			breakIndex, breakPrice, action, multiTPs, updatedEntryData := checkTrendBreak(&latestEntryData, relCandleIndex, relCandleIndex, candles)

			if updatedEntryData.MultiTPs[0].Price > 0.0 {
				latestEntryData = updatedEntryData
			}

			// if relCandleIndex > 100 && relCandleIndex < 300 {
			// 	fmt.Printf(colorCyan+"<%v> strat1 latestEntryData= %+v\n", relCandleIndex, latestEntryData.MultiTPs)
			// }

			if breakIndex > 0 && breakPrice > 0 && action != "MULTI-TP" {
				breakTrend(candles, breakIndex, relCandleIndex, &newLabels, &latestEntryData)
				stored.Trades = append(stored.Trades, latestEntryData)
				(*strategy).CloseLong(breakPrice, 100, -1, relCandleIndex, action, candles[len(candles)-1], bot)
			} else if breakIndex > 0 && action == "MULTI-TP" {
				// if relCandleIndex < 3000 {
				// 	for _, p := range multiTPs {
				// 		fmt.Printf(colorYellow+"<%v> %+v\n"+colorReset, relCandleIndex, p)
				// 	}
				// }

				if len(multiTPs) > 0 && multiTPs[0].Price > 0 {
					for _, tpPoint := range multiTPs {
						if tpPoint.Order == tpPoint.TotalPointsInSet {
							// fmt.Printf(colorGreen+"<%v> BREAK TREND point= %+v\n latestEntry= %+v\n", relCandleIndex, tpPoint, latestEntryData)
							breakTrend(candles, breakIndex, relCandleIndex, &newLabels, &latestEntryData)
							stored.Trades = append(stored.Trades, latestEntryData) //TODO: how to append trade when not all TPs hit?
						}
						(*strategy).CloseLong(tpPoint.Price, -1, tpPoint.CloseSize, relCandleIndex, action, candles[len(candles)-1], bot)
					}
				}
			}

			if updatedEntryData.MultiTPs[0].Price > 0.0 {
				stored.Trades[len(stored.Trades)-1] = latestEntryData //entry data will be updated if multi TP
			}
		} else {
			// fmt.Printf(colorCyan+"<%v> SEARCH new entry\n", relCandleIndex)
			possibleEntryIndexes := pivotWatchEntryCheck(low, stored.PivotLows, pivotLowsToEnter, 0)

			if len(possibleEntryIndexes) > 0 {
				//check if latest possible entry eligible
				var lastTradeExitIndex int
				if len(stored.Trades) == 0 {
					lastTradeExitIndex = 0
				} else {
					lastTradeExitIndex = stored.Trades[len(stored.Trades)-1].BreakIndex
				}

				//latest entry PL must be 1) after last trade end, and 2) be the latest PL
				latestPossibleEntry := possibleEntryIndexes[len(possibleEntryIndexes)-1]
				minTradingIndex := 0
				if len(latestActions) > 0 && latestActions[0].Action == "SL" {
					minTradingIndex = (lastTradeExitIndex + slCooldownCandles)
				} else if len(latestActions) > 0 && latestActions[0].Action == "MULTI-TP" {
					minTradingIndex = (lastTradeExitIndex + tpCooldownCandles)
				} else {
					minTradingIndex = lastTradeExitIndex
				}

				//time cannot be within block window
				timeOK := true
				if tradeWindowStart != "" && tradeWindowEnd != "" {
					et, _ := time.Parse(httpTimeFormat, strings.Split(candles[latestPossibleEntry].TimeOpen, ".")[0])
					s, _ := time.Parse("15:04:05", tradeWindowStart)
					e, _ := time.Parse("15:04:05", tradeWindowEnd)

					afterS := true
					if et.Hour() > s.Hour() {
						afterS = true
					} else if et.Hour() == s.Hour() {
						if et.Minute() > s.Minute() {
							afterS = true
						} else {
							afterS = false
						}
					} else {
						afterS = false
					}

					beforeE := true
					if et.Hour() < e.Hour() {
						beforeE = true
					} else if et.Hour() == e.Hour() {
						if et.Minute() < e.Minute() {
							beforeE = true
						} else {
							beforeE = false
						}
					} else {
						beforeE = false
					}

					timeOK = afterS && beforeE

					// if relCandleIndex > 400 && relCandleIndex < 1400 {
					// 	fmt.Printf(colorGreen+"<%v> OK= %v/ et= %v,%v / s= %v,%v (%v) / e= %v,%v (%v)\n", relCandleIndex, timeOK, et.Hour(), et.Minute(), s.Hour(), s.Minute(), afterS, e.Hour(), e.Minute(), beforeE)
					// }
				}

				if latestPossibleEntry > minTradingIndex && latestPossibleEntry == stored.PivotLows[len(stored.PivotLows)-1] && timeOK {
					newEntryData := StrategyDataPoint{}
					newEntryData = logEntry(relCandleIndex, pivotLowsToEnter, latestPossibleEntry, candles, possibleEntryIndexes, stored.PivotLows, stored.Trades, &newEntryData, &newLabels, maxDurationCandles, 1-(slPerc/100), -1, -1, -1, tpMap)
					newEntryData.ActualEntryIndex = relCandleIndex

					// if relCandleIndex < 300 {
					// 	fmt.Printf(colorCyan+"<%v> ENTER possibleEntries= %v \n newEntryData=%+v\n", relCandleIndex, possibleEntryIndexes, newEntryData)
					// }

					//enter long
					completedMultiTPs := (*strategy).Buy(close[relCandleIndex], newEntryData.SLPrice, newEntryData.TPPrice, newEntryData.StartTrailPerc, newEntryData.TrailingPerc, risk, int(lev), relCandleIndex, newEntryData.MultiTPs, candles[len(candles)-1], true, bot)
					newEntryData.MultiTPs = completedMultiTPs
					// fmt.Printf("<%v> %+v\n", relCandleIndex, newEntryData.MultiTPs)

					stored.Trades = append(stored.Trades, newEntryData)
				}
			}
		}
	}

	// if relCandleIndex < 250 && relCandleIndex > 120 {
	// 	fmt.Printf(colorRed+"<%v> pl=%v\nph=%v\n"+colorReset, relCandleIndex, stored.PivotLows, stored.PivotHighs)
	// }
	*storage = stored
	return newLabels, 0
}

// SCANNING //

type PivotTrendScanStore struct {
	PivotHighs    []int
	PivotLows     []int
	ScanPoints    []StrategyDataPoint
	WatchingTrend bool
}

func logEntry(relCandleIndex, pivotLowsToEnter, entryIndex int, candles []Candlestick, possibleEntryPLIndexes, allPivotLows []int, dataPoints []StrategyDataPoint, retData *StrategyDataPoint, newLabels *(map[string]map[int]string), maxDurationCandles int, slPerc, tpPerc, startTrailPerc, trailingPerc float64, tpMap map[float64]float64) StrategyDataPoint {
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

		if len(allPivotLows) >= pivotLowsToEnter {
			plSliIndexOfEntryPL := 0
			for i := 0; i < len(allPivotLows); i++ {
				if allPivotLows[i] == entryIndex {
					plSliIndexOfEntryPL = i
					break
				}
			}
			if (plSliIndexOfEntryPL-pivotLowsToEnter-1) >= 0 && plSliIndexOfEntryPL < len(allPivotLows) {
				firstEntryPivotIndex := allPivotLows[plSliIndexOfEntryPL-(pivotLowsToEnter-1)]
				firstEntryPivot := candles[firstEntryPivotIndex].Low
				lastEntryPivotIndex := allPivotLows[plSliIndexOfEntryPL]
				lastEntryPivot := candles[lastEntryPivotIndex].Low

				// if relCandleIndex < 5000 {
				// 	fmt.Printf(colorGreen+"first= %v / last= %v \n"+colorReset, firstEntryPivotIndex, lastEntryPivotIndex)
				// }

				retData.FirstToLastEntryPivotDuration = lastEntryPivotIndex - firstEntryPivotIndex
				retData.FirstLastEntryPivotPriceDiffPerc = ((firstEntryPivot - lastEntryPivot) / firstEntryPivot) * 100

				priceDiffPercTotal := 0.0
				for i := 0; i < pivotLowsToEnter-1; i++ {
					// if relCandleIndex < 5000 {
					// 	fmt.Printf(colorCyan+"<%v> plSliIndexOfEntryPL= %v (%v) / pl1= %v / pl2 = %v\n __ %v\n"+colorReset, relCandleIndex, plSliIndexOfEntryPL, allPivotLows[plSliIndexOfEntryPL], allPivotLows[plSliIndexOfEntryPL-i], allPivotLows[plSliIndexOfEntryPL-i-1], allPivotLows)
					// }

					pl1 := candles[allPivotLows[plSliIndexOfEntryPL-i]].Low
					pl2 := candles[allPivotLows[plSliIndexOfEntryPL-i-1]].Low
					priceDiffPercTotal = priceDiffPercTotal + math.Abs(((pl2-pl1)/pl2)*100)
				}
				retData.AveragePriceDiffPercEntryPivots = priceDiffPercTotal / float64(pivotLowsToEnter)
			}
		}

		retData.MaxExitIndex = actualEntryIndex + maxDurationCandles

		retData.SLPrice = slPerc * candles[relCandleIndex].Close
		if tpMap != nil {
			retData.MultiTPs = []MultiTPPoint{}

			//sort map in ascending order of price
			keys := make([]float64, 0, len(tpMap))
			for k := range tpMap {
				keys = append(keys, k)
			}
			sort.Slice(keys, func(i, j int) bool {
				return keys[i] < keys[j]
			})
			//convert to struct
			for i, profitPerc := range keys {
				tpPrice := retData.EntryTradeOpenCandle.Close * (1 + (profitPerc / 100))
				retData.MultiTPs = append(retData.MultiTPs, MultiTPPoint{
					Order:            i + 1,
					IsDone:           false,
					Price:            tpPrice,
					ClosePerc:        tpMap[profitPerc],
					TotalPointsInSet: len(tpMap),
				})
			}
		}

		if tpPerc > 0 {
			retData.TPPrice = tpPerc * candles[relCandleIndex].Close
		}
		if startTrailPerc > 0 {
			retData.StartTrailPerc = startTrailPerc
		}
		if trailingPerc > 0 {
			retData.TrailingPerc = trailingPerc
		}

		(*newLabels)["middle"][relCandleIndex-actualEntryIndex] = fmt.Sprintf(">/%v", retData.ActualEntryIndex)
	}

	// fmt.Printf(colorYellow+"<%v> retData= %+v\n"+colorReset, retData.EntryTradeOpenCandle.DateTime(), retData)
	return *retData
}

func checkTrendBreak(entryData *StrategyDataPoint, relCandleIndex, startCheckIndex int, candles []Candlestick) (int, float64, string, []MultiTPPoint, StrategyDataPoint) {
	// if relCandleIndex < 2100 && relCandleIndex > 1550 {
	// 	fmt.Printf(colorPurple+"<%v> checkSL sl= %v / startCheckIndex= %v / entryData= %+v\n", relCandleIndex, slPrice, startCheckIndex, entryData)
	// }

	//check max index
	if relCandleIndex >= entryData.MaxExitIndex && entryData.MaxExitIndex != 0 {
		return relCandleIndex, candles[relCandleIndex].Close, "MAX", nil, *entryData
	}

	//check SL + TP
	for i := startCheckIndex; i <= relCandleIndex; i++ {
		//sl
		if candles[i].Low <= entryData.SLPrice && entryData.SLPrice > 0 {
			return i, entryData.SLPrice, "SL", nil, *entryData
		}

		//tp
		if candles[i].High >= entryData.TPPrice && entryData.TPPrice > 0 {
			return i, entryData.TPPrice, "TP", nil, *entryData
		}

		//multi-tp (map)
		updatedTPs := []MultiTPPoint{}
		if entryData.MultiTPs != nil {
			// if relCandleIndex > 570 && relCandleIndex < 600 {
			// 	fmt.Printf("%+v\n", entryData.MultiTPs)
			// }

			retTPPoints := []MultiTPPoint{}
			for _, tpPoint := range entryData.MultiTPs {
				p := MultiTPPoint{}

				if tpPoint.IsDone {
					continue
				}

				if tpPoint.Price > 0.0 && candles[i].High >= tpPoint.Price && !tpPoint.IsDone {
					// fmt.Printf(colorYellow+"<%v> TRIGGERED multi TP / high= %v / tpPoint= %+v\n", i, candles[i].High, tpPoint)

					p = tpPoint
					p.IsDone = true
					//add price to exit price slice (in case multiple TPs)
					retTPPoints = append(retTPPoints, p)
				}

				updatedTPs = append(updatedTPs, p)
				// if len(retTPPoints) > 0 {
				// 	fmt.Printf(colorCyan+"%+v\n", retTPPoints)
				// }
			}

			// if relCandleIndex > 570 && relCandleIndex < 600 {
			// 	fmt.Printf(colorPurple+"updated TPs= %+v\n"+colorReset, updatedTPs)
			// }

			if len(updatedTPs) > 0 && updatedTPs[0].Price > 0 {
				// fmt.Printf(colorPurple+"updated TPs= %+v\n"+colorReset, updatedTPs)
				newTPPoints := []MultiTPPoint{}
				for _, exTP := range entryData.MultiTPs {
					//look for updated version oif tp point
					toAdd := MultiTPPoint{}
					for _, up := range updatedTPs {
						if up.Order == exTP.Order {
							toAdd = up
							break
						}
					}
					if toAdd.Price > 0.0 {
						newTPPoints = append(newTPPoints, toAdd)
					} else {
						newTPPoints = append(newTPPoints, exTP)
					}
				}
				(*entryData).MultiTPs = newTPPoints
				// fmt.Printf(colorYellow+"(*entryData).MultiTPs= %+v\n"+colorReset, (*entryData).MultiTPs)
			}

			// if relCandleIndex > 570 && relCandleIndex < 600 {
			// 	fmt.Printf(colorYellow+"(*entryData).MultiTPs= %+v\n"+colorReset, (*entryData).MultiTPs)
			// }

			if len(retTPPoints) <= 0 {
				return i, -1, "MULTI-TP", retTPPoints, (*entryData)
			} else {
				return i, retTPPoints[len(retTPPoints)-1].Price, "MULTI-TP", retTPPoints, (*entryData)
			}
		}

		//trailingTP
		// if relCandleIndex < 150 && relCandleIndex > 100 {
		// 	fmt.Printf(colorRed+"<%v> %+v\n"+colorReset, relCandleIndex, entryData)
		// }

		if entryData.StartTrailPerc > 0 && entryData.TrailingPerc > 0 {
			if entryData.TrailingStarted {
				//adjust trailing min + max
				if candles[i].High > entryData.TrailingMax {
					(*entryData).TrailingMax = candles[i].High
				}

				//check if hit trailing stop
				trailStopoutPrice := (1 - (entryData.TrailingPerc / 100)) * entryData.TrailingMax
				if candles[i].Low <= trailStopoutPrice {
					return i, trailStopoutPrice, "TRAIL", nil, *entryData
				}
			} else {
				//check if should activate trailing stop
				startTrailPrice := candles[entryData.ActualEntryIndex].Close * (1 + (entryData.StartTrailPerc / 100))
				if candles[i].High >= startTrailPrice {
					(*entryData).TrailingStarted = true
					(*entryData).TrailingMax = candles[i].High //only track trailing max for strategy simulate, trailing min only needed for scanning purposes

					// if relCandleIndex < 200 {
					// 	fmt.Printf(colorGreen+"<%v> TRAIL_STOP(%v) triggered @ $%v \n > %+v\n\n"+colorReset, relCandleIndex, startTrailPrice, candles[i].High, entryData)
					// }
				}
			}
		}
	}

	return -1, -1.0, "", nil, StrategyDataPoint{}
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

	//trailing tp data log
	startTrailPrice := (1 + (retData.StartTrailPerc / 100)) * candles[retData.ActualEntryIndex].Close
	trailingStarted := false
	trailingMaxDrawdownPerc := -1.0
	for i := retData.ActualEntryIndex; i <= trendExtentIndex; i++ {
		//only start logging data once trailing started
		if !trailingStarted && candles[i].High >= startTrailPrice {
			trailingStarted = true
		}

		if trailingStarted {
			if retData.TrailingMax <= 0.0 || candles[i].High > retData.TrailingMax {
				(*retData).TrailingMax = candles[i].High
			}
			if retData.TrailingMin <= 0.0 || candles[i].Low > retData.TrailingMin {
				(*retData).TrailingMin = candles[i].Low
			}

			trailingMaxDrawdownPerc = ((retData.TrailingMax - retData.TrailingMin) / retData.TrailingMax) * 100
			if (retData.TrailingMaxDrawdownPercTillExtent <= 0.0 || trailingMaxDrawdownPerc > retData.TrailingMaxDrawdownPercTillExtent) && (trailingMaxDrawdownPerc > 0) {
				retData.TrailingMaxDrawdownPercTillExtent = trailingMaxDrawdownPerc
			}
		}
	}
	// if relCandleIndex < 600 {
	// 	fmt.Printf(colorGreen+"<%v> startTrail= %v / TrailingMax= %v / TrailingMin= %v / newTrailingMaxDraw= %v / STORED maxDrawdownTrail= %v\n __ entry= %v (%v)\n ___ extentPerc(GROWTH)= %v\n / entryTime= %v / extentTime= %v (%v) / duration= %v\n\n"+colorReset,
	// 		relCandleIndex, startTrailPrice, retData.TrailingMax, retData.TrailingMin, trailingMaxDrawdownPerc, retData.TrailingMaxDrawdownPercTillExtent, retData.EntryTradeOpenCandle.Close, retData.ActualEntryIndex, fmt.Sprintf("%.2f", retData.Growth), entryTime, trendExtentTime, trendExtentIndex, retData.Duration)
	// }

	// fmt.Printf(colorRed+"<%v> retData= %+v\n"+colorReset, retData.BreakTime, retData)
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
	//map of profit % TO account size perc to close (multi-tp)
	tpMap := map[float64]float64{
		1.3: 10,
		1.9: 20,
		2.5: 20,
		3.3: 20,
		3.9: 25,
		5.7: 5,
	}

	pivotLowsToEnter := 6
	maxDurationCandles := 1200
	slPerc := 1.0
	slCooldownCandles := 35
	// tpCooldownCandles := 35

	// tradeWindowStart := "09:00:00"
	tradeWindowStart := ""
	// tradeWindowEnd := "18:00:00"
	tradeWindowEnd := ""

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
		"top":    map[int]string{},
		"middle": map[int]string{},
		"bottom": map[int]string{},
	}

	//calculate pivots
	newLabels, _ = findPivots(open, high, low, close, relCandleIndex, &(stored.PivotHighs), &(stored.PivotLows), newLabels)
	// newLabels["middle"][0] = fmt.Sprintf("%v", relCandleIndex)

	if len(stored.ScanPoints) > 0 && relCandleIndex <= stored.ScanPoints[len(stored.ScanPoints)-1].BreakIndex+slCooldownCandles {
		newLabels["middle"][0] = "ч"

		// if relCandleIndex > 80 && relCandleIndex < 200 {
		// 	fmt.Printf(colorCyan+"<%v> %+v\n"+colorReset, candles[len(candles)-1].DateTime(), stored.ScanPoints[len(stored.ScanPoints)-1])
		// }
	} else if len(stored.PivotLows) >= 4 {
		if stored.WatchingTrend {
			//manage pos
			// fmt.Printf(colorYellow+"checking existing trend %v %v\n"+colorReset, relCandleIndex, candles[len(candles)-1].DateTime)
			latestEntryData := stored.ScanPoints[len(stored.ScanPoints)-1]

			//check sl + tp + max duration
			breakIndex, breakPrice, action, _, updatedEntryData := checkTrendBreak(&latestEntryData, relCandleIndex, relCandleIndex, candles)

			if updatedEntryData.MultiTPs[0].Price > 0.0 {
				latestEntryData = updatedEntryData
			}

			// if relCandleIndex > 100 && relCandleIndex < 300 {
			// 	fmt.Printf(colorCyan+"<%v> strat1 latestEntryData= %+v\n", relCandleIndex, latestEntryData.MultiTPs)
			// }

			if breakIndex > 0 && breakPrice > 0 && action != "MULTI-TP" {
				breakTrend(candles, breakIndex, relCandleIndex, &newLabels, &latestEntryData)
				stored.ScanPoints = append(stored.ScanPoints, latestEntryData)
				stored.WatchingTrend = false
				retData = latestEntryData
			}

			if updatedEntryData.MultiTPs[0].Price > 0.0 {
				stored.ScanPoints[len(stored.ScanPoints)-1] = latestEntryData //entry data will be updated if multi TP
			}
		} else {
			// fmt.Printf(colorCyan+"<%v> SEARCH new entry\n", relCandleIndex)
			possibleEntryIndexes := pivotWatchEntryCheck(low, stored.PivotLows, pivotLowsToEnter, 0)

			if len(possibleEntryIndexes) > 0 {
				//check if latest possible entry eligible
				var lastTradeExitIndex int
				if len(stored.ScanPoints) == 0 {
					lastTradeExitIndex = 0
				} else {
					lastTradeExitIndex = stored.ScanPoints[len(stored.ScanPoints)-1].BreakIndex
				}

				//latest entry PL must be 1) after last trade end, and 2) be the latest PL
				latestPossibleEntry := possibleEntryIndexes[len(possibleEntryIndexes)-1]
				minTradingIndex := lastTradeExitIndex + slCooldownCandles

				//time cannot be within block window
				timeOK := true
				if tradeWindowStart != "" && tradeWindowEnd != "" {
					et, _ := time.Parse(httpTimeFormat, strings.Split(candles[latestPossibleEntry].TimeOpen, ".")[0])
					s, _ := time.Parse("15:04:05", tradeWindowStart)
					e, _ := time.Parse("15:04:05", tradeWindowEnd)

					afterS := true
					if et.Hour() > s.Hour() {
						afterS = true
					} else if et.Hour() == s.Hour() {
						if et.Minute() > s.Minute() {
							afterS = true
						} else {
							afterS = false
						}
					} else {
						afterS = false
					}

					beforeE := true
					if et.Hour() < e.Hour() {
						beforeE = true
					} else if et.Hour() == e.Hour() {
						if et.Minute() < e.Minute() {
							beforeE = true
						} else {
							beforeE = false
						}
					} else {
						beforeE = false
					}

					timeOK = afterS && beforeE

					// if relCandleIndex > 400 && relCandleIndex < 1400 {
					// 	fmt.Printf(colorGreen+"<%v> OK= %v/ et= %v,%v / s= %v,%v (%v) / e= %v,%v (%v)\n", relCandleIndex, timeOK, et.Hour(), et.Minute(), s.Hour(), s.Minute(), afterS, e.Hour(), e.Minute(), beforeE)
					// }
				}

				if latestPossibleEntry > minTradingIndex && latestPossibleEntry == stored.PivotLows[len(stored.PivotLows)-1] && timeOK {
					newEntryData := StrategyDataPoint{}
					newEntryData = logEntry(relCandleIndex, pivotLowsToEnter, latestPossibleEntry, candles, possibleEntryIndexes, stored.PivotLows, stored.ScanPoints, &newEntryData, &newLabels, maxDurationCandles, 1-(slPerc/100), -1, -1, -1, tpMap)
					newEntryData.ActualEntryIndex = relCandleIndex
					stored.ScanPoints = append(stored.ScanPoints, newEntryData)
					stored.WatchingTrend = true

					// if relCandleIndex < 300 {
					// 	fmt.Printf(colorCyan+"<%v> ENTER possibleEntries= %v \n newEntryData=%+v\n", relCandleIndex, possibleEntryIndexes, newEntryData)
					// }
				}
			}
		}
	}

	*storage = stored
	return newLabels, retData
}
