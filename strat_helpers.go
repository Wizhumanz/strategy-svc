package main

import (
	"math"
)

func checkExists(val int, slice []int) bool {
	found := false
	for _, v := range slice {
		if v == val {
			found = true
			break
		}
	}
	return found
}

// findPivots returns a labels map and a boolean value that is true if the current candle is a pivot low
func findPivots(
	open, high, low, close []float64,
	relCandleIndex int,
	ph, pl *[]int,
	labels map[string]map[int]string) (map[string]map[int]string, bool) {
	foundPL := false
	// fmt.Printf(colorWhite+"findPivots index %v | o = %v, h = %v, l = %v, c = %v\n"+colorReset, relCandleIndex, len(open), len(high), len(low), len(close))

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

	pivotBarsBack := 0
	var newPivotSearchStartIndex int
	if len(*ph) == 0 && len(*pl) == 0 {
		newPivotSearchStartIndex = 0
	} else if len(*ph) == 0 {
		newPivotSearchStartIndex = (*pl)[len(*pl)-1]
	} else if len(*pl) == 0 {
		newPivotSearchStartIndex = (*ph)[len(*ph)-1]
	} else {
		newPivotSearchStartIndex = int(math.Max(float64((*ph)[len(*ph)-1]), float64((*pl)[len(*pl)-1])))
		newPivotSearchStartIndex = int(math.Max(float64(1), float64(newPivotSearchStartIndex))) //make sure index is at least 1 to subtract 1 later
		newPivotSearchStartIndex++                                                              //don't allow both pivot high and low on same candle
	}

	// if relCandleIndex > 127 && relCandleIndex < 170 {
	// 	fmt.Printf(colorGreen+"<%v> lookHigh= %v / searchStartIndex= %v\n ph(%v)=%v \n pl(%v)= %v\n"+colorReset, relCandleIndex, lookForHigh, newPivotSearchStartIndex, len(*ph), *ph, len(*pl), *pl)
	// }

	if lookForHigh {
		// fmt.Println(colorRed + "looking for HIGH" + colorReset)
		//check if new candle took out the low of previous candles since last pivot
		for i := newPivotSearchStartIndex; (i+1) < len(low) && (i+1) < len(high); i++ { //TODO: should be relCandleIndex-1 but causes index outta range err
			if (low[i+1] < low[i]) && (high[i+1] < high[i]) {
				//check if pivot already exists
				found := checkExists(i, *ph)
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

				if newPHIndex >= 0 && !(checkExists(newPHIndex, *ph)) {
					// fmt.Printf("Appending PH %v\n", newPHIndex)

					*ph = append(*ph, newPHIndex)
					pivotBarsBack = relCandleIndex - newPHIndex

					labels["top"][pivotBarsBack] = "H"
					// pivotBarsBack: fmt.Sprintf("H from %v", relCandleIndex),
					break
				}
			}
		}
	} else {
		// fmt.Println(colorYellow + "looking for LOW" + colorReset)
		for i := newPivotSearchStartIndex; (i+1) < len(high) && (i+1) < len(low); i++ {
			// if relCandleIndex > 127 && relCandleIndex < 170 {
			// 	fmt.Printf(colorPurple+"<%v> checking PL %v\n", relCandleIndex, i)
			// }

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

				//find lowest low since last PL
				newPLIndex := i
				// if relCandleIndex > 127 && relCandleIndex < 170 {
				// 	fmt.Printf(colorYellow+"<%v, %v> new PL init index = %v\n"+colorReset, relCandleIndex, close[relCandleIndex], newPLIndex)
				// }

				//find actual lowest point since last PH to label as PL
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
						// if relCandleIndex > 127 && relCandleIndex < 170 {
						// 	fmt.Printf(colorRed+"<%v, %v> skipping add = %v\n"+colorReset, relCandleIndex, close[relCandleIndex], newPLIndex)
						// }
						continue
					}
				}

				if newPLIndex >= 0 {
					*pl = append(*pl, newPLIndex)
					pivotBarsBack = relCandleIndex - newPLIndex
					labels["bottom"][pivotBarsBack] = "L"
					// pivotBarsBack: fmt.Sprintf("L%v", relCandleIndex),

					foundPL = true
					break
				}
			}
		}
	}

	return labels, foundPL
}
