package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
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
	EntryDateTime string  `json:"EntryDateTime"`
	ExitDateTime  string  `json:"ExitDateTime"`
	Direction     string  `json:"Direction"`
	EntryPrice    float64 `json:"EntryPrice"`
	ExitPrice     float64 `json:"ExitPrice"`
	PosSize       float64 `json:"PosSize"`
	RiskedEquity  float64 `json:"RiskedEquity"`
	Profit        float64 `json:"Profit"`
	RawProfitPerc float64 `json:"RawProfitPerc"`
	TotalFees     float64 `json:"TotalFees"`
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
	Action       string
	Price        float64
	SL           float64
	PosSize      float64
	RiskedEquity float64
	ProfitCap    float64
	ExchangeFee  float64
	DateTime     string
}

type StrategyExecutor struct {
	posLongSize     float64
	posShortSize    float64
	totalEquity     float64
	availableEquity float64
	Actions         map[int]StrategyExecutorAction //map bar index to action that occured at that index
	liveTrade       bool
	lastEntryEquity float64

	OrderSlippagePerc    float64
	ExchangeTradeFeePerc float64
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

func calcMultiTPs(multiTPs []MultiTPPoint, actualPosSize float64, index int) []MultiTPPoint {
	retMultiTPs := []MultiTPPoint{}
	if len(multiTPs) > 0 && multiTPs[0].Price > 0 {
		calcRemainingPosSize := actualPosSize
		for i, tpPoint := range multiTPs {
			// fmt.Printf(colorCyan+"<%v, %v> actualPosSz= %v / remainingSz = %v / tpPoint = %+v / multiTPs= %+v\n", index, i, actualPosSize, calcRemainingPosSize, tpPoint, multiTPs)

			newPoint := tpPoint
			if i == len(multiTPs)-1 {
				//make sure to close entire position (account for ultra-low leftover size from calculations)
				newPoint.CloseSize = calcRemainingPosSize
			} else {
				newPoint.CloseSize = (tpPoint.ClosePerc / 100) * actualPosSize
				calcRemainingPosSize -= newPoint.CloseSize
			}
			retMultiTPs = append(retMultiTPs, newPoint)
		}
	}
	return retMultiTPs
}

// Buy returns a []MultiTPPoint with actual position sizes for each TP point based on actual entry data
func (strat *StrategyExecutor) Buy(price, sl, tp, startTrailPerc, trailingPerc, accRisk float64, lev, cIndex int, multiTPs []MultiTPPoint, candle Candlestick, directionIsLong bool, botStreamName Bot) []MultiTPPoint {
	retMultiTPs := []MultiTPPoint{}

	if !strat.liveTrade {
		actualPrice := (1 + (strat.OrderSlippagePerc / 100)) * price //TODO: modify to - for shorting
		desiredPosCap, _ := calcEntry(actualPrice, sl, accRisk, strat.availableEquity, lev)
		//binance min order size = 10 USDT
		if desiredPosCap <= 10 {
			return retMultiTPs
		}
		actualCap := (1 - (strat.ExchangeTradeFeePerc / 100)) * desiredPosCap
		exchangeFee := (strat.ExchangeTradeFeePerc / 100) * desiredPosCap
		actualPosSize := actualCap / actualPrice
		strat.availableEquity = strat.availableEquity - actualCap
		strat.totalEquity = strat.availableEquity + actualCap
		strat.lastEntryEquity = actualCap

		// fmt.Printf(colorGreen+"actualPrice= %v (%v)\nsl= %v\ntp= %v\nactualCap= (%v)%v\nactualPosSize= %v\n --> $%v\n"+colorReset, actualPrice, price, sl, tp, actualCap, desiredPosCap, actualPosSize, strat.totalEquity)

		if directionIsLong {
			strat.posLongSize = actualPosSize
		} else {
			strat.posShortSize = actualPosSize
		}

		//complete multi-tp map with actual pos sizes
		retMultiTPs = calcMultiTPs(multiTPs, actualPosSize, cIndex)

		strat.Actions[cIndex] = StrategyExecutorAction{
			Action:       "ENTER",
			Price:        actualPrice,
			SL:           sl,
			PosSize:      actualPosSize,
			RiskedEquity: (accRisk / 100) * strat.availableEquity,
			ExchangeFee:  exchangeFee,
			DateTime:     candle.DateTime(),
		}
	} else {
		//TODO: call calcMultiTPs and use result as multi tp prices and order sizes

		// get acc balance
		var objmap []map[string]interface{}
		if err := json.Unmarshal(getFuturesAccountBalance(), &objmap); err != nil {
			log.Fatal(err)
		}

		var binanceSymbolsFile []map[string]interface{}
		file, _ := ioutil.ReadFile("./json-data/symbols-binance-fut-perp.json")
		if err := json.Unmarshal(file, &binanceSymbolsFile); err != nil {
			log.Fatal(err)
		}

		var symbol string
		for _, s := range binanceSymbolsFile {
			if s["symbol_id"] == botStreamName.Ticker {
				symbol = s["symbol_id_exchange"].(string)
			}
		}

		changeMarginType(symbol)
		changeInitialLeverage(symbol, lev)
		cancelAllOpenOrders(symbol)

		var balance string
		for _, b := range objmap {
			if b["asset"] == "USDT" {
				balance = b["balance"].(string)
			}
		}

		// calculate pos size (20% of account size)
		bal, _ := strconv.ParseFloat(balance, 64)
		currentBalance := bal * float64(lev) * 0.2

		//TODO: use TP map to submit multi TP stop limit orders
		// retMultiTPs = calcMultiTPs(multiTPs, actualPosSize, cIndex)

		// submit 3 orders:
		// 1. stop limit order SL (stop=0.8*price, limit=0.79*price, reduceOnly=true)
		newOrder(symbol, "SELL", "STOP", fmt.Sprintf("%.2f", currentBalance/(0.8*price)), fmt.Sprintf("%.2f", 0.79*price), "true", fmt.Sprintf("%.2f", 0.8*price))
		// 2. stop limit order TP (stop=1.5*price, limit=1.49*price, reduceOnly=true)
		newOrder(symbol, "SELL", "TAKE_PROFIT", fmt.Sprintf("%.2f", currentBalance/(0.8*price)), fmt.Sprintf("%.2f", 1.49*price), "true", fmt.Sprintf("%.2f", 1.5*price))
		newOrder(symbol, "SELL", "TAKE_PROFIT", fmt.Sprintf("%.2f", currentBalance/(0.8*price)), fmt.Sprintf("%.2f", 1.49*price), "true", fmt.Sprintf("%.2f", 1.5*price))
		// 3. limit order entry (limit=0.8*price)
		newOrder(symbol, "BUY", "LIMIT", fmt.Sprintf("%.2f", currentBalance/(0.8*price)), fmt.Sprintf("%.2f", 0.8*price), "no", "0")

		// args := map[string]interface{}{}
		// args["slPrice"] = float64(sl)
		// args["accRisk"] = float64(accRisk)
		// args["leverage"] = int(lev)
		// args["latestClosePrice"] = float64(price)
		// pauseStreamListening(botStreamName, fmt.Sprintf("OpenTradeSaga | %v", args))
		// OpenTradeSaga.Execute(botStreamName, svcConsumerGroupName, redisConsumerID, args)
		// continueStreamListening(botStreamName)
	}

	// startTrailPrice := price * (1 + (startTrailPerc / 100))

	// fmt.Printf(colorYellow+"<%v> BUYING $=%v / sl=%v / tpMap= %+v \n len(strat.Actions)= %v\n\n"+colorReset, cIndex, price, sl, retMultiTPs, len(strat.Actions))

	// for _, action := range strat.Actions {
	// 	fmt.Printf("%+v\n", action)
	// }

	return retMultiTPs
}

func (strat *StrategyExecutor) CloseLong(price, posPercToClose, closeSz float64, cIndex int, action string, candle Candlestick, bot Bot) {
	// fmt.Printf(colorRed+"<%v> CLOSING TRADE(%v) $= %v / posPercClose= %v / closeSz= %v \n"+colorReset, cIndex, action, price, posPercToClose, closeSz)

	if !strat.liveTrade {
		orderSize := 0.0
		if closeSz > 0.0 {
			orderSize = closeSz
		} else {
			orderSize = (posPercToClose / 100) * strat.posLongSize
		}
		actualClosePrice := (1 - (strat.OrderSlippagePerc / 100)) * price //TODO: modify to + for shorting
		actualCloseCap := (1 - (strat.ExchangeTradeFeePerc / 100)) * (actualClosePrice * orderSize)
		exchangeFee := (strat.ExchangeTradeFeePerc / 100) * (actualClosePrice * orderSize)

		strat.availableEquity = strat.availableEquity + actualCloseCap
		strat.posLongSize = strat.posLongSize - orderSize
		strat.totalEquity = strat.availableEquity + (strat.posLongSize * price) //run this line on every iteration to constantly update equity (including unrealized PnL)

		// fmt.Printf(colorYellow+"CLOSE actualPrice= %v (%v)\nactualCloseEquity= %v\norderSz = %v\n --> $%v\n"+colorReset, actualClosePrice, price, actualCloseCap, orderSize, strat.totalEquity)

		// _, file, line, _ := runtime.Caller(0)
		// Log(fmt.Sprintf("<%v> SIM closed pos %v/100 at %v | action = %v\n ---> $%v", cIndex, posPercToClose, price, action, strat.totalEquity),
		// 	fmt.Sprintf("<%v> %v", line, file))

		strat.Actions[cIndex] = StrategyExecutorAction{
			Action:      action,
			Price:       actualClosePrice,
			PosSize:     orderSize,
			ExchangeFee: exchangeFee,
			ProfitCap:   actualCloseCap - strat.lastEntryEquity,
			DateTime:    candle.DateTime(),
		}
	} else {
		_, file, line, _ := runtime.Caller(0)
		go Log(fmt.Sprintf("Closing pos %v/100 at %v | action = %v\n", posPercToClose, price, action),
			fmt.Sprintf("<%v> %v", line, file))

		//NOTE: current strategies don't need active order closing

		// args := map[string]interface{}{}
		// args["posPercToClose"] = posPercToClose
		// args["ticker"] = bot.Ticker
		// pauseStreamListening(bot.KEY, fmt.Sprintf("ExitTradeSaga | %v", args))
		// ExitTradeSaga.Execute(bot.KEY, svcConsumerGroupName, redisConsumerID, args)
		// continueStreamListening(bot.KEY)
	}
}
