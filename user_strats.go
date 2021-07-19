package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"strings"
	"time"
)

//return signature: (label, bars back to add label, storage obj to pass to next func call/iteration)
func strat1(
	candles []Candlestick, risk, lev, accSz float64,
	open, high, low, close []float64,
	relCandleIndex int,
	strategy *StrategyExecutor,
	storage *interface{}, bot Bot, pivotLowsNum, maxDurationNum, slCooldown, tpCooldown int, slPercent, tpSingle float64) (map[string]map[int]string, int) {
	//TODO: pass these 2 from frontend
	strategy.OrderSlippagePerc = 0.15
	strategy.ExchangeTradeFeePerc = 0.075

	// //map of profit % TO account size perc to close (multi-tp)
	// tpMap := map[float64]float64{
	// 	3.5: 20, //second largest
	// 	3.8: 10, //largest
	// 	4.0: 70,
	// }

	// 4 PL to enter

	//map of profit % TO account size perc to close (multi-tp)
	// tpMap := map[float64]float64{
	// 	1.5: 20,
	// 	3.0: 10,
	// 	3.5: 70,
	// }

	// pivotLowsToEnter := 4
	// maxDurationCandles := 480
	// slPerc := 1.0
	// slCooldownCandles := 35
	// tpCooldownCandles := 0

	// tradeWindows := []ValRange{
	// 	{
	// 		Start: "00:00:00",
	// 		End:   "00:00:48",
	// 	},
	// 	{
	// 		Start: "00:04:48",
	// 		End:   "00:06:24",
	// 	},
	// 	{
	// 		Start: "00:08:48",
	// 		End:   "00:10:24",
	// 	},
	// 	{
	// 		Start: "00:16:00",
	// 		End:   "00:22:24",
	// 	},
	// 	{
	// 		Start: "00:23:12",
	// 		End:   "00:23:59",
	// 	},
	// }

	tpMap := map[float64]float64{
		// 1.5: 70,
		tpSingle: 100,
	}

	pivotLowsToEnter := pivotLowsNum
	maxDurationCandles := maxDurationNum
	slPerc := slPercent
	slCooldownCandles := slCooldown
	tpCooldownCandles := tpCooldown

	tradeWindows := []ValRange{
		// {
		// 	Start: "09:00:00",
		// 	End:   "11:00:00",
		// },
		// {
		// 	Start: "15:38:00",
		// 	End:   "17:00:00",
		// },
		// {
		// 	Start: "16:00:00",
		// 	End:   "18:24:00",
		// },
	}

	// entryPivotNoTradeZones := []ValRange{
	// 	{
	// 		Start: 0.0,
	// 		End:   0.72,
	// 	},
	// 	{
	// 		Start: 0.8,
	// 		End:   0.88,
	// 	},
	// 	{
	// 		Start: 0.96,
	// 		End:   2.16,
	// 	},
	// 	{
	// 		Start: 2.24,
	// 		End:   999.99,
	// 	},
	// }

	entryPivotTradeZones := []ValRange{}

	newLabels := map[string]map[int]string{
		"top":    map[int]string{},
		"middle": map[int]string{},
		"bottom": map[int]string{},
	}
	// newLabels["middle"][0] = fmt.Sprintf("%v", relCandleIndex)

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

			if len(updatedEntryData.MultiTPs) > 0 && updatedEntryData.MultiTPs[0].Price > 0.0 {
				latestEntryData = updatedEntryData
			}

			// if relCandleIndex > 100 && relCandleIndex < 300 {
			// 	fmt.Printf(colorCyan+"<%v> strat1 latestEntryData= %+v\n", relCandleIndex, latestEntryData.MultiTPs)
			// }

			if breakIndex > 0 && breakPrice > 0 && action != "MULTI-TP" {
				// fmt.Printf(colorYellow+"%v %v (%v)\n"+colorReset, action, breakPrice, breakIndex)

				breakTrend(candles, breakIndex, relCandleIndex, &newLabels, &latestEntryData, action)
				stored.Trades = append(stored.Trades, latestEntryData)
				(*strategy).CloseLong(breakPrice, 100, -1, relCandleIndex, action, candles[len(candles)-1], bot)
			} else if breakIndex > 0 && action == "MULTI-TP" {
				// if relCandleIndex < 3000 {
				// 	for _, p := range multiTPs {
				// 		fmt.Printf(colorYellow+"<%v> MULTI-TP %+v\n"+colorReset, relCandleIndex, p)
				// 	}
				// }

				if len(multiTPs) > 0 && multiTPs[0].Price > 0 {
					for _, tpPoint := range multiTPs {
						if tpPoint.Order == tpPoint.TotalPointsInSet {
							// fmt.Printf(colorGreen+"<%v> BREAK TREND point= %+v\n latestEntry= %+v\n", relCandleIndex, tpPoint, latestEntryData)
							breakTrend(candles, breakIndex, relCandleIndex, &newLabels, &latestEntryData, action)
							stored.Trades = append(stored.Trades, latestEntryData) //TODO: how to append trade when not all TPs hit?
						}
						(*strategy).CloseLong(tpPoint.Price, -1, tpPoint.CloseSize, relCandleIndex, action, candles[len(candles)-1], bot)
					}
				}
			}

			if len(updatedEntryData.MultiTPs) > 0 && updatedEntryData.MultiTPs[0].Price > 0.0 {
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
				timeOK := false
				if len(tradeWindows) <= 0 {
					timeOK = true
				}
				et, _ := time.Parse(httpTimeFormat, strings.Split(candles[latestPossibleEntry].TimeOpen, ".")[0])
				for _, window := range tradeWindows {
					s, _ := time.Parse("15:04:05", window.Start.(string))
					e, _ := time.Parse("15:04:05", window.End.(string))

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

					if timeOK {
						break
					}
					// if relCandleIndex < 1400 {
					// 	fmt.Printf(colorGreen+"<%v> OK= %v/ et= %v,%v / s= %v,%v (%v) / e= %v,%v (%v)\n", relCandleIndex, timeOK, et.Hour(), et.Minute(), s.Hour(), s.Minute(), afterS, e.Hour(), e.Minute(), beforeE)
					// }
				}

				//entry pivots price diff cannot be within block windows
				entryPivotsDiffOK := false
				lastPLIndex := latestPossibleEntry
				lastPL := candles[lastPLIndex].Low
				firstPLIndex := stored.PivotLows[len(stored.PivotLows)-1-(pivotLowsToEnter-1)]
				firstPL := candles[firstPLIndex].Low
				var entryPivotsPriceDiffPerc float64 = math.Abs(((firstPL - lastPL) / firstPL) * 100)
				// fmt.Printf(colorYellow+"<%v> %v / %v\n"+colorReset, relCandleIndex, entryPivotsPriceDiffPerc, entryPivotTradeZones)
				for _, window := range entryPivotTradeZones {
					if entryPivotsPriceDiffPerc >= (window.Start.(float64)/100) && entryPivotsPriceDiffPerc <= (window.End.(float64)/100) {
						entryPivotsDiffOK = true
						break
					}
				}
				if len(entryPivotTradeZones) <= 0 {
					entryPivotsDiffOK = true
				}

				//random trade selector
				rand.Seed(time.Now().UnixNano())
				r := rand.Intn(40)
				randOK := r == 1
				randOK = true

				if latestPossibleEntry > minTradingIndex && latestPossibleEntry == stored.PivotLows[len(stored.PivotLows)-1] && timeOK && entryPivotsDiffOK && randOK {
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

	if len(stored.PivotHighs)%(pivotLowsToEnter+1) == 0 && len(stored.PivotLows)%(pivotLowsToEnter+1) != 0 {
		return newLabels, pivotLowsToEnter*2 - (len(stored.PivotHighs) % (pivotLowsToEnter + 1)) - 0
	}
	return newLabels, pivotLowsToEnter*2 - (len(stored.PivotHighs) % (pivotLowsToEnter + 1)) - (len(stored.PivotLows) % (pivotLowsToEnter + 1))
}
