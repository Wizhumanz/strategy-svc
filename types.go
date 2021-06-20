package main

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
)

type jsonResponse struct {
	Msg  string `json:"message"`
	Body string `json:"body"`
}

//for unmarshalling JSON to bools
type JSONBool bool

func (bit *JSONBool) UnmarshalJSON(b []byte) error {
	txt := string(b)
	*bit = JSONBool(txt == "1" || txt == "true")
	return nil
}

type User struct {
	K           *datastore.Key `datastore:"__key__"`
	Name        string         `json:"name"`
	Email       string         `json:"email"`
	AccountType string         `json:"type"`
	Password    string         `json:"password"`
}

func (l User) String() string {
	r := ""
	v := reflect.ValueOf(l)
	typeOfL := v.Type()

	for i := 0; i < v.NumField(); i++ {
		r = r + fmt.Sprintf("%s: %v, ", typeOfL.Field(i).Name, v.Field(i).Interface())
	}
	return r
}

type TradeAction struct {
	KEY         string  `json:"KEY"`
	UserID      string  `json:"UserID"`
	Action      string  `json:"Action"`
	AggregateID int     `json:"AggregateID,string"`
	BotID       string  `json:"BotID"`
	Direction   string  `json:"Direction"` //LONG or SHORT
	Size        float32 `json:"Size,string"`
	Timestamp   string  `json:"Timestamp"`
	Ticker      string  `json:"Ticker"`
	Exchange    string  `json:"Exchange"`
	OrderType   int     `json:"OrderType"`
}

// API types
type createCheckoutSessionResponse struct {
	SessionID string `json:"id"`
}

type loginReq struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type webHookRequest struct {
	User            string `json:"User"`
	Ticker          string `json:"Ticker"`
	Direction       string `json:"Direction"`
	TradeActionType string `json:"TradeActionType"` // ENTER, EXIT, SL, TP
	Size            string `json:"Size"`
}

type Bot struct {
	KEY                     string         `json:"KEY"`
	K                       *datastore.Key `datastore:"__key__"`
	Name                    string         `json:"Name"`
	AggregateID             int            `json:"AggregateID,string"`
	UserID                  string         `json:"UserID"`
	ExchangeConnection      string         `json:"ExchangeConnection"`
	AccountRiskPercPerTrade string         `json:"AccountRiskPercPerTrade"`
	AccountSizePercToTrade  string         `json:"AccountSizePercToTrade"`
	IsActive                bool           `json:"IsActive,string"`
	IsArchived              bool           `json:"IsArchived,string"`
	Leverage                string         `json:"Leverage"`
	Timestamp               string         `json:"Timestamp"`
	Ticker                  string         `json:"Ticker"`
	Period                  string         `json:"Period"`
	WebhookConnectionID     string         `json:"WebhookConnectionID"`
	CreationDate            string         `json:"CreationDate"`
}

func (l Bot) String() string {
	r := ""
	v := reflect.ValueOf(l)
	typeOfL := v.Type()

	for i := 0; i < v.NumField(); i++ {
		r = r + fmt.Sprintf("%s: %v, ", typeOfL.Field(i).Name, v.Field(i).Interface())
	}
	return r
}

type ExchangeConnection struct {
	K         *datastore.Key `datastore:"__key__"`
	KEY       string         `json:"KEY"`
	Name      string         `json:"Name"`
	APIKey    string         `json:"APIKey"`
	UserID    string         `json:"UserID"`
	IsDeleted bool           `json:"IsDeleted,string"`
	Timestamp string         `json:"Timestamp"`
}

type WebhookConnection struct {
	K           *datastore.Key `datastore:"__key__"`
	KEY         string         `json:"KEY"`
	URL         string         `json:"URL"`
	Name        string         `json:"Name"`
	Description string         `json:"Description"`
	IsPublic    bool           `json:"IsPublic,string"`
}

func (l WebhookConnection) String() string {
	r := ""
	v := reflect.ValueOf(l)
	typeOfL := v.Type()

	for i := 0; i < v.NumField(); i++ {
		r = r + fmt.Sprintf("%s: %v, ", typeOfL.Field(i).Name, v.Field(i).Interface())
	}
	return r
}

type ScatterData struct {
	Profit   float64 `json:"Profit"`
	Duration float64 `json:"Duration"`
	Size     int     `json:"Size"`
	Leverage int     `json:"Leverage"`
	Time     int     `json:"Time"`
}

type upwardTrend struct {
	// EntryTime  string  `json:"EntryTime"`
	// ExtentTime string  `json:"ExtentTime"`
	Duration int     `json:"Duration"`
	Growth   float64 `json:"Growth"`
}

type CoinAPITicker struct {
	ID         string `json:"symbol_id"`
	ExchangeID string `json:"symbol_id_exchange"`
	BaseAsset  string `json:"asset_id_base"`
	QuoteAsset string `json:"asset_id_quote"`
}

type WebsocketPacket struct {
	ResultID string        `json:"ResultID"`
	Data     []interface{} `json:"Data"`
}

type CandlestickChartData struct {
	DateTime        string  `json:"DateTime"`
	Open            float64 `json:"Open"`
	High            float64 `json:"High"`
	Low             float64 `json:"Low"`
	Close           float64 `json:"Close"`
	StratEnterPrice float64 `json:"StratEnterPrice"`
	StratExitPrice  float64 `json:"StratExitPrice"`
	LabelTop        string  `json:"LabelTop"`
	LabelMiddle     string  `json:"LabelMiddle"`
	LabelBottom     string  `json:"LabelBottom"`
}

type ComputeRequest struct {
	Operation        string `json:"operation"`
	Ticker           string `json:"ticker"`
	Period           string `json:"period"`
	TimeStart        string `json:"time_start"`
	TimeEnd          string `json:"time_end"`
	CandlePacketSize string `json:"candlePacketSize"`
	User             string `json:"user"`
	Risk             string `json:"risk"`
	Leverage         string `json:"leverage"`
	Size             string `json:"size"`
}

type ShareResult struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	ResultFileName string `json:"resultFileName"`
	ShareID        string `json:"shareID"`
	UserID         string `json:"userID"`
}

type ProfitCurveDataPoint struct {
	DateTime string  `json:"DateTime"`
	Equity   float64 `json:"Equity"`
}

type ProfitCurveData struct {
	Label string                 `json:"DataLabel"`
	Data  []ProfitCurveDataPoint `json:"Data"`
}

type SimulatedTradeDataPoint struct {
	DateTime      string  `json:"DateTime"`
	Direction     string  `json:"Direction"`
	EntryPrice    float64 `json:"EntryPrice"`
	ExitPrice     float64 `json:"ExitPrice"`
	PosSize       float64 `json:"PosSize"`
	RiskedEquity  float64 `json:"RiskedEquity"`
	RawProfitPerc float64 `json:"RawProfitPerc"`
}

type SimulatedTradeData struct {
	Label string                    `json:"DataLabel"`
	Data  []SimulatedTradeDataPoint `json:"Data"`
}

type BacktestResFile struct {
	Ticker               string                 `json:"Ticker"`
	Period               string                 `json:"Period"`
	Start                string                 `json:"Start"`
	End                  string                 `json:"End"`
	ModifiedCandlesticks []CandlestickChartData `json:"ModifiedCandlesticks"`
	ProfitCurve          []ProfitCurveData      `json:"ProfitCurve"`
	SimulatedTrades      []SimulatedTradeData   `json:"SimulatedTrades"`
}

type Candlestick struct {
	// DateTime    string
	PeriodStart string  `json:"time_period_start"`
	PeriodEnd   string  `json:"time_period_end"`
	TimeOpen    string  `json:"time_open"`
	TimeClose   string  `json:"time_close"`
	Open        float64 `json:"price_open"`
	High        float64 `json:"price_high"`
	Low         float64 `json:"price_low"`
	Close       float64 `json:"price_close"`
	Volume      float64 `json:"volume_traded"`
	TradesCount float64 `json:"trades_count"`
}

func (c *Candlestick) Create(redisData map[string]string) {
	c.Open, _ = strconv.ParseFloat(redisData["open"], 32)
	c.High, _ = strconv.ParseFloat(redisData["high"], 32)
	c.Low, _ = strconv.ParseFloat(redisData["low"], 32)
	c.Close, _ = strconv.ParseFloat(redisData["close"], 32)
	c.Volume, _ = strconv.ParseFloat(redisData["volume"], 32)
	c.TradesCount, _ = strconv.ParseFloat(redisData["tradesCount"], 32)
	c.TimeOpen = redisData["timeOpen"]
	c.TimeClose = redisData["timeClose"]
	c.PeriodStart = redisData["periodStart"]
	c.PeriodEnd = redisData["periodEnd"]
	// t, timeErr := time.Parse(httpTimeFormat, strings.Split(redisData["periodStart"], ".")[0])
	// if timeErr != nil {
	// 	fmt.Errorf("&v", timeErr)
	// 	return
	// }
	// c.DateTime = t.Format(httpTimeFormat)
}

func (c *Candlestick) DateTime() string {
	t, timeErr := time.Parse(httpTimeFormat, strings.Split(c.PeriodStart, ".")[0])
	if timeErr != nil {
		fmt.Errorf("&v", timeErr)
		return "Error with DateTime"
	}
	return t.Format(httpTimeFormat)
}

type StrategyExecutorAction struct {
	Action  string
	Price   float64
	SL      float64
	PosSize float64
}

type StrategyExecutor struct {
	posLongSize     float64
	posShortSize    float64
	totalEquity     float64
	availableEquity float64
	Actions         map[int]StrategyExecutorAction //map bar index to action that occured at that index
	liveTrade       bool
}

func (strat *StrategyExecutor) Init(e float64, liveTrade bool) {
	strat.totalEquity = e
	strat.availableEquity = e
	strat.Actions = map[int]StrategyExecutorAction{}
	strat.liveTrade = liveTrade
}

func (strat *StrategyExecutor) GetLiveTradeStatus() bool {
	return strat.liveTrade
}

func (strat *StrategyExecutor) GetTotalEquity() float64 {
	return strat.totalEquity
}

func (strat *StrategyExecutor) GetAvailableEquity() float64 {
	return strat.availableEquity
}

func (strat *StrategyExecutor) GetPosLongSize() float64 {
	return strat.posLongSize
}

func (strat *StrategyExecutor) Buy(price, sl, tp, accRisk float64, lev, cIndex int, directionIsLong bool, botStreamName string) {
	if !strat.liveTrade {
		_, posSize := calcEntry(price, sl, accRisk, strat.availableEquity, lev)

		strat.availableEquity = strat.availableEquity - (posSize * price)

		if directionIsLong {
			strat.posLongSize = posSize
		} else {
			strat.posShortSize = posSize
		}

		strat.Actions[cIndex] = StrategyExecutorAction{
			Action:  "ENTER",
			Price:   price,
			SL:      sl,
			PosSize: posSize,
		}
	} else {
		// _, file, line, _ := runtime.Caller(0)
		// go Log(loggingInJSON(fmt.Sprintf("[%v] OpenTradeSaga simulated START", cIndex)),
		// 	fmt.Sprintf("<%v> %v", line, file))

		//TODO: replace with binance func
		args := map[string]interface{}{}
		args["slPrice"] = float64(sl)
		args["accRisk"] = float64(accRisk)
		args["leverage"] = int(lev)
		args["latestClosePrice"] = float64(price)
		pauseStreamListening(botStreamName, fmt.Sprintf("OpenTradeSaga | %v", args))
		OpenTradeSaga.Execute(botStreamName, svcConsumerGroupName, redisConsumerID, args)
		continueStreamListening(botStreamName)
	}
}

func (strat *StrategyExecutor) CloseLong(price, posPercToClose float64, cIndex int, action string, timestamp string, bot Bot) {
	if !strat.liveTrade {
		orderSize := (posPercToClose / 100) * strat.posLongSize
		strat.availableEquity = strat.availableEquity + (orderSize * price)
		strat.posLongSize = strat.posLongSize - orderSize
		strat.totalEquity = strat.availableEquity + (strat.posLongSize * price)

		strat.Actions[cIndex] = StrategyExecutorAction{
			Action:  action,
			Price:   price,
			PosSize: orderSize,
		}
	} else {
		_, file, line, _ := runtime.Caller(0)
		go Log(fmt.Sprintf("Closing pos %v/100 at %v | action = %v\n", posPercToClose, price, action),
			fmt.Sprintf("<%v> %v", line, file))

		//TODO: replace with binance func
		args := map[string]interface{}{}
		args["posPercToClose"] = posPercToClose
		args["ticker"] = bot.Ticker
		pauseStreamListening(bot.KEY, fmt.Sprintf("ExitTradeSaga | %v", args))
		ExitTradeSaga.Execute(bot.KEY, svcConsumerGroupName, redisConsumerID, args)
		continueStreamListening(bot.KEY)
	}
}
