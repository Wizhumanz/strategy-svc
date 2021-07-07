package main

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// calcEntry returns (posCap, posSize) so that SL only loses fixed percentage of account
func calcEntry(entryPrice, slPrice, accPercRisk, accSz float64, leverage int) (float64, float64) {
	rawRiskPerc := (entryPrice - slPrice) / entryPrice
	if rawRiskPerc < 0 {
		return -1, -1
	}

	accRisk := (accPercRisk / 100) * accSz
	posRisk := (rawRiskPerc)
	leveragedPosEquity := accRisk / posRisk

	if leveragedPosEquity > accSz*float64(leverage) {
		leveragedPosEquity = accSz * float64(leverage)
	}
	posSize := leveragedPosEquity / entryPrice

	fmt.Printf(colorGreen+"entry= %v, SL= %v (%v), posEq= %v \n accSz x lev= %v / accPercRisk= %v / accSz= %v / lev= %v / rawRisk= %v\n"+colorReset, entryPrice, slPrice, ((entryPrice-slPrice)/entryPrice)*100, leveragedPosEquity, accSz*float64(leverage), accPercRisk, accSz, leverage, rawRiskPerc)

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

func breakTrend(candles []Candlestick, breakIndex, relCandleIndex int, newLabels *(map[string]map[int]string), retData *StrategyDataPoint, action string) {
	(*retData).BreakIndex = breakIndex
	(*retData).BreakTime = candles[breakIndex].DateTime()
	(*retData).EndAction = action

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
