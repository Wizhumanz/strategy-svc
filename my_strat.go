package main

import (
	"encoding/json"
	"fmt"
	"runtime"
)

var alreadyInTrade bool = false
var preEmas []float64

func myStrat(
	candles []Candlestick, risk, lev, accSz float64,
	open, high, low, close []float64,
	relCandleIndex int,
	strategy *StrategyExecutor,
	storage *interface{}, bot Bot,
	smas []float64,
	emas []float64,
	volumeAverage []float64,
	volatility, volumeIndex float64,
) (map[string]map[int]string, int, map[string]string, bool, float64) {
	tradeIsLong := false
	strategy.OrderSlippagePerc = 0.15
	strategy.ExchangeTradeFeePerc = 0.075

	// tpSingle := 3.0
	// tpMap := map[float64]float64{
	// 	tpSingle: 100,
	// }

	// pivotLowsToEnter := 5
	// maxDurationCandles := 300
	slPerc := 2.0
	// slCooldownCandles := 0
	// tpCooldownCandles := 0

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

	if len(emas) > 3 && len(preEmas) > 3 && preEmas[0] >= preEmas[1] && emas[0] <= emas[1] && alreadyInTrade {
		newLabels["middle"][0] = "X"
		(*strategy).CloseLong(candles[relCandleIndex].Close, 100, -1, relCandleIndex, "SELL", candles[len(candles)-1], bot, tradeIsLong)
		alreadyInTrade = false
	}

	if len(emas) > 3 && len(preEmas) > 3 && preEmas[0] <= preEmas[1] && emas[0] >= emas[1] {
		newLabels["middle"][0] = "E"
		_ = (*strategy).Buy(close[relCandleIndex], candles[relCandleIndex].Close*1-(slPerc/100), 0, 0, 0, risk, int(lev), relCandleIndex, nil, candles[len(candles)-1], tradeIsLong, bot)
		alreadyInTrade = true
	}

	if len(emas) >= 4 {
		preEmas = emas
	}

	return newLabels, 0, nil, true, 0
}
