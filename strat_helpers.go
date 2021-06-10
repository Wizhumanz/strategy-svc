package main

import "math"

func findPivots(
	open, high, low, close []float64,
	relCandleIndex int,
	ph, pl *[]int) (map[string]map[int]string, bool) {
	foundPL := false

	//find pivot highs + lows
	var lookForHigh bool
	if len(*ph) == 1 && len(*pl) == 0 {
		lookForHigh = false
	} else if len(*ph) == 0 && len(*pl) == 0 {
		lookForHigh = true
	} else if (*ph)[len(*ph)-1] < (*pl)[len(*pl)-1] {
		lookForHigh = true
	} else {
		lookForHigh = false
	}
	newLabels := make(map[string]map[int]string) //map of labelPos:map of labelBarsBack:labelText
	// newLabels["middle"] = map[int]string{
	// 	0: fmt.Sprintf("%v", relCandleIndex),
	// }

	pivotBarsBack := 0
	var lastPivotIndex int
	if len(*ph) == 0 && len(*pl) == 0 {
		lastPivotIndex = 0
	} else if len(*ph) == 0 {
		lastPivotIndex = (*pl)[len(*pl)-1]
	} else if len(*pl) == 0 {
		lastPivotIndex = (*ph)[len(*ph)-1]
	} else {
		lastPivotIndex = int(math.Max(float64((*ph)[len(*ph)-1]), float64((*pl)[len(*pl)-1])))
		lastPivotIndex = int(math.Max(float64(1), float64(lastPivotIndex))) //make sure index is at least 1 to subtract 1 later
		lastPivotIndex++                                                    //don't allow both pivot high and low on same candle
	}
	if lookForHigh {
		// fmt.Println(colorRed + "looking for HIGH" + colorReset)
		//check if new candle took out the low of previous candles since last pivot
		for i := lastPivotIndex; (i+1) < len(low) && (i+1) < len(high); i++ { //TODO: should be relCandleIndex-1 but causes index outta range err
			if (low[i+1] < low[i]) && (high[i+1] < high[i]) {
				//check if pivot already exists
				found := false
				for _, ph := range *ph {
					if ph == i {
						found = true
						break
					}
				}
				if found {
					continue
				}

				//find highest high since last PL
				newPHIndex := i
				if len(*pl) > 0 && len(*ph) > 0 && newPHIndex > 0 {
					latestPLIndex := (*pl)[len(*pl)-1]
					latestPHIndex := (*ph)[len(*ph)-1]
					for f := newPHIndex - 1; f >= latestPLIndex && f > latestPHIndex; f-- {
						if high[f] > high[newPHIndex] && !found {
							newPHIndex = f
						}
					}

					//check if current candle actually clears new selected candle as pivot high
					if !((low[i+1] < low[newPHIndex]) && (high[i+1] < high[newPHIndex])) {
						continue
					}
				}

				if newPHIndex >= 0 {
					*ph = append(*ph, newPHIndex)
					pivotBarsBack = relCandleIndex - newPHIndex

					newLabels["top"] = map[int]string{
						// pivotBarsBack: fmt.Sprintf("H from %v", relCandleIndex),
						pivotBarsBack: "H",
					}
				}

				break
			}
		}
	} else {
		// fmt.Println(colorYellow + "looking for LOW" + colorReset)
		for i := lastPivotIndex; (i+1) < len(high) && (i+1) < len(low); i++ {
			if (high[i+1] > high[i]) && (low[i+1] > low[i]) {
				//check if pivot already exists
				found := false
				for _, pl := range *pl {
					if pl == i {
						found = true
						break
					}
				}
				if found {
					continue
				}
				// fmt.Printf("Found PL at index %v", j)

				//find lowest low since last PL
				newPLIndex := i
				// if relCandleIndex > 150 && relCandleIndex < 170 {
				// 	fmt.Printf(colorYellow+"<%v> new PL init index = %v\n"+colorReset, relCandleIndex, newPLIndex)
				// }

				if len(*ph) > 0 && len(*pl) > 0 && newPLIndex > 0 {
					latestPHIndex := (*ph)[len(*ph)-1]
					latestPLIndex := (*pl)[len(*pl)-1]
					// if relCandleIndex > 150 && relCandleIndex < 170 {
					// 	fmt.Printf("SEARCH lowest low latestPHIndex = %v, latestPLIndex = %v\n", latestPHIndex, latestPLIndex)
					// }
					for f := newPLIndex - 1; f >= latestPHIndex && f > latestPLIndex && f < len(low) && f < len(high); f-- {
						if low[f] < low[newPLIndex] && !found {
							newPLIndex = f
						}
					}

					//check if current candle actually clears new selected candle as pivot high
					if !((high[i+1] > high[newPLIndex]) && (low[i+1] > low[newPLIndex])) {
						continue
					}
				}

				if newPLIndex >= 0 {
					*pl = append(*pl, newPLIndex)
					pivotBarsBack = relCandleIndex - newPLIndex
					newLabels["bottom"] = map[int]string{
						// pivotBarsBack: fmt.Sprintf("L from %v", relCandleIndex),
						pivotBarsBack: "L",
					}
					foundPL = true
					// fmt.Printf("Adding PL index %v\n", newPLIndex)
				}

				break
			}
		}
	}

	return newLabels, foundPL
}
