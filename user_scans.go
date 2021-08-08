package main

import (
	"fmt"
	"math"
	"strings"
	"time"
)

func scanPivotTrends(
	candles []Candlestick,
	open, high, low, close []float64,
	relCandleIndex int,
	storage *interface{}) (map[string]map[int]string, StrategyDataPoint) {

	//NOTE: tp map not used in scanning
	tpMap := map[float64]float64{
		99.0: 100,
	}

	tradeIsLong := true
	pivotLowsToEnter := 4
	maxDurationCandles := 1000
	slPerc := 3.0
	slCooldownCandles := 35
	// tpCooldownCandles := 35

	tradeWindows := []ValRange{
		// {
		// 	Start: "06:36:00",
		// 	End:   "07:48:00",
		// },
		// {
		// 	Start: "08:24:00",
		// 	End:   "09:00:00",
		// },
	}

	entryPivotNoTradeZones := []ValRange{}

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
			breakIndex, breakPrice, action, _, updatedEntryData := checkTrendBreak(&latestEntryData, relCandleIndex, relCandleIndex, candles, tradeIsLong)

			if len(updatedEntryData.MultiTPs) > 0 && updatedEntryData.MultiTPs[0].Price > 0.0 {
				latestEntryData = updatedEntryData
			}

			// if relCandleIndex > 100 && relCandleIndex < 300 {
			// 	fmt.Printf(colorCyan+"<%v> strat1 latestEntryData= %+v\n", relCandleIndex, latestEntryData.MultiTPs)
			// }

			if breakIndex > 0 && breakPrice > 0 && action != "MULTI-TP" {
				breakTrend(candles, breakIndex, relCandleIndex, &newLabels, &latestEntryData, action, tradeIsLong)
				stored.ScanPoints = append(stored.ScanPoints, latestEntryData)
				stored.WatchingTrend = false
				retData = latestEntryData
			}

			if len(updatedEntryData.MultiTPs) > 0 && updatedEntryData.MultiTPs[0].Price > 0.0 {
				stored.ScanPoints[len(stored.ScanPoints)-1] = latestEntryData //entry data will be updated if multi TP
			}
		} else {
			// fmt.Printf(colorCyan+"<%v> SEARCH new entry\n", relCandleIndex)
			possibleEntryIndexes := pivotWatchEntryCheck(low, stored.PivotLows, pivotLowsToEnter, 0, tradeIsLong)

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
				entryPivotsDiffOK := true
				lastPLIndex := latestPossibleEntry
				lastPL := candles[lastPLIndex].Low
				firstPLIndex := stored.PivotLows[len(stored.PivotLows)-1-(pivotLowsToEnter-1)]
				firstPL := candles[firstPLIndex].Low
				var entryPivotsPriceDiffPerc float64 = math.Abs(((firstPL - lastPL) / firstPL) * 100)
				for _, window := range entryPivotNoTradeZones {
					if entryPivotsPriceDiffPerc > (window.Start.(float64)/100) && entryPivotsPriceDiffPerc < (window.End.(float64)/100) {
						entryPivotsDiffOK = false
						break
					}
				}

				if latestPossibleEntry > minTradingIndex && latestPossibleEntry == stored.PivotLows[len(stored.PivotLows)-1] && timeOK && entryPivotsDiffOK {
					newEntryData := StrategyDataPoint{}
					newEntryData = logEntry(relCandleIndex, pivotLowsToEnter, latestPossibleEntry, candles, possibleEntryIndexes, stored.PivotLows, stored.ScanPoints, &newEntryData, &newLabels, maxDurationCandles, 1-(slPerc/100), -1, -1, -1, tpMap, tradeIsLong)
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
