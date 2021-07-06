package main

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
	EndAction                         string         `json:EndAction`
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

type PivotTrendScanStore struct {
	PivotHighs    []int
	PivotLows     []int
	ScanPoints    []StrategyDataPoint
	WatchingTrend bool
}
