package main

import (
	"fmt"
	"runtime"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"gitlab.com/myikaco/msngr"
	"gitlab.com/myikaco/saga"
)

var OpenTradeSaga saga.Saga = saga.Saga{
	Steps: []saga.SagaStep{
		{
			Transaction:             calcPosSize,
			CompensatingTransaction: cancelCalcPosSize,
		},
		{
			Transaction:             submitEntryOrder,
			CompensatingTransaction: cancelSubmitEntryOrder,
		},
	},
}

// OpenTradeSaga T1
func calcPosSize(allArgs ...interface{}) (interface{}, error) {
	fmt.Println("SAGA: Running calcPosSize")

	transactionArgs := allArgs[0].(map[string]interface{})
	funcArgs := allArgs[1].(map[string]interface{})
	fmt.Printf("Inside OpenTradeSaga step, args = %v\n", funcArgs)

	//get acc balance for pos size calc (send msg to order-svc)
	msgs := []string{}
	msgs = append(msgs, "Calc")
	msgs = append(msgs, "GetBal")
	msgs = append(msgs, "Asset")
	msgs = append(msgs, "USDT")
	msgs = append(msgs, "BotStreamName")
	msgs = append(msgs, transactionArgs["botStream"].(string))
	msngr.AddToStream(transactionArgs["botStream"].(string), msgs)

	//listen for msg resp
	listenArgs := make(map[string]string)
	listenArgs["streamName"] = transactionArgs["botStream"].(string)
	listenArgs["groupName"] = svcConsumerGroupName
	listenArgs["consumerName"] = redisConsumerID
	listenArgs["start"] = ">"
	listenArgs["count"] = "1"

	var bal string
	parserHandlers := []msngr.CommandHandler{
		{
			Command: "Bal",
			HandlerMatches: []msngr.HandlerMatch{
				{
					Matcher: func(fieldVal string) bool {
						return fieldVal != ""
					},
					Handler: func(msg redis.XMessage, output *interface{}) {
						bal = msngr.FilterMsgVals(msg, func(k, v string) bool {
							return (k == "Bal" && v != "")
						})

						if bal != "" {
							_, file, line, _ := runtime.Caller(0)
							go Log(loggingInJSON(fmt.Sprintf("OpenTradeSaga get bal = %v <%v>", bal, time.Now().UTC().Format(httpTimeFormat))),
								fmt.Sprintf("<%v> %v", line, file))
						}
					},
				},
			},
		},
	}
	msngr.ReadAndParse(msngr.ReadStream, "OpenTradeSaga calcPosSize", msngr.ParseStream, listenArgs, parserHandlers)

	//calc pos size
	accBal, _ := strconv.ParseFloat(bal, 32)

	// Set to long by default
	_, posSize := calcEntry(funcArgs["latestClosePrice"].(float64), funcArgs["slPrice"].(float64), funcArgs["accRisk"].(float64), accBal, funcArgs["leverage"].(int), true)

	return posSize, nil
}

// OpenTradeSaga T-1
func cancelCalcPosSize(allArgs ...interface{}) (interface{}, error) {
	fmt.Println("SAGA: Running cancelCalcPosSize")
	// nothing to cancel
	return nil, nil
}

// OpenTradeSaga T2
func submitEntryOrder(allArgs ...interface{}) (interface{}, error) {
	fmt.Println("SAGA: Running submitEntryOrder")

	transactionArgs := allArgs[0].(map[string]interface{})

	// XADD submitEntryOrderIntent
	msgs := []string{}
	msgs = append(msgs, "Action")
	msgs = append(msgs, "SubmitEntryOrderIntent")
	msgs = append(msgs, "Symbol")
	msgs = append(msgs, "BTCUSDT")
	msgs = append(msgs, "Side")
	msgs = append(msgs, "BUY")
	msgs = append(msgs, "Quantity")
	msgs = append(msgs, "1000")
	msgs = append(msgs, "Price")
	msgs = append(msgs, "69")
	msgs = append(msgs, "Timestamp")
	msgs = append(msgs, time.Now().Format("2006-01-02_15:04:05_-0700"))
	msngr.AddToStream(transactionArgs["botStream"].(string), msgs)

	//listen for msg resp
	listenArgs := make(map[string]string)
	listenArgs["streamName"] = transactionArgs["botStream"].(string)
	listenArgs["groupName"] = svcConsumerGroupName
	listenArgs["consumerName"] = redisConsumerID
	listenArgs["start"] = ">"
	listenArgs["count"] = "1"
	var order string
	parserHandlers := []msngr.CommandHandler{
		{
			Command: "Entry Order",
			HandlerMatches: []msngr.HandlerMatch{
				{
					Matcher: func(fieldVal string) bool {
						return fieldVal != ""
					},
					Handler: func(msg redis.XMessage, output *interface{}) {
						order = msngr.FilterMsgVals(msg, func(k, v string) bool {
							return (k == "Entry Order" && v != "")
						})
						fmt.Println(order)
					},
				},
			},
		},
	}
	msngr.ReadAndParse(msngr.ReadStream, "strat-svc submitEntryOrder consec headers retrieve", msngr.ParseStream, listenArgs, parserHandlers)

	//listen for consec responses
	msngr.ListenConsecResponses(transactionArgs, "strat-svc submitEntryOrder ListenConsecResponses", func(i int, v string, m redis.XMessage, isHeaderMatch bool) {
		fmt.Printf("Read consec header at index %v val: %s, IsMatch = %v (%s)", i, v, isHeaderMatch, m.ID)
	})

	// order-svc:
	//  entryOrderSubmitted, entryOrderFilled
	//  entryOrderFailed
	//  entryOrderSubmitted, entryOrderFilled, SLExitedTrade/TPExitedTrade

	_, file, line, _ := runtime.Caller(0)
	go Log(loggingInJSON(fmt.Sprintf("! OPENTRADE SAGA COMPLETE | args = %v", allArgs...)),
		fmt.Sprintf("<%v> %v", line, file))
	return nil, nil
}

// OpenTradeSaga T-2
func cancelSubmitEntryOrder(allArgs ...interface{}) (interface{}, error) {
	fmt.Println("SAGA: Running cancelSubmitEntryOrder")

	// XADD cancelEntryOrderIntent {timestamp}

	// order-svc:
	//  entryOrderCancelled
	return nil, nil
}

// stop loss and take profit (maybe partial exits), and full exit
var ExitTradeSaga saga.Saga = saga.Saga{
	Steps: []saga.SagaStep{
		{
			Transaction:             calcCloseSize,
			CompensatingTransaction: cancelCalcCloseSize,
		},
		{
			Transaction:             submitExitOrder,
			CompensatingTransaction: cancelSubmitExitOrder,
		},
	},
}

// OpenTradeSaga T1
func calcCloseSize(allArgs ...interface{}) (interface{}, error) {
	fmt.Println("SAGA: Running calcCloseSize")

	transactionArgs := allArgs[0].(map[string]interface{})
	funcArgs := allArgs[1].(map[string]interface{})
	fmt.Printf("Inside ExitTradeSaga step, args = %v\n", funcArgs)

	//get pos size for pos size calc (send msg to order-svc)
	msgs := []string{}
	msgs = append(msgs, "Calc")
	msgs = append(msgs, "GetPosSize")
	msgs = append(msgs, "Ticker")
	msgs = append(msgs, funcArgs["ticker"].(string))
	msngr.AddToStream(transactionArgs["botStream"].(string), msgs)
	time.Sleep(1 * time.Second)

	//listen for msg resp
	listenArgs := make(map[string]string)
	listenArgs["streamName"] = transactionArgs["botStream"].(string)
	listenArgs["groupName"] = svcConsumerGroupName
	listenArgs["consumerName"] = redisConsumerID
	listenArgs["start"] = ">"
	listenArgs["count"] = "1"

	var posSize string
	for {
		_, msg, err := msngr.ReadStream(listenArgs, "OpenTradeSaga calcCloseSize")
		fmt.Println(colorGreen + "Finished ReadStream" + colorReset)
		if err != nil {
			_, file, line, _ := runtime.Caller(0)
			go Log(loggingInJSON(fmt.Sprintf("CalcPosSize saga step ReadStream err = %v", err)),
				fmt.Sprintf("<%v> %v", line, file))
			return nil, err
		}

		if str, ok := msg.([]redis.XStream); ok {
			posSize = msngr.FilterMsgVals(str[0].Messages[0], func(k, v string) bool {
				return k == "PosSize" && v != ""
			})
		}

		//calc exit size
		if posSize == "" {
			// return 0, fmt.Errorf("posSize calc result empty %v", allArgs...)
			fmt.Printf("posSize calc result empty %v", allArgs...)
		} else {
			break
		}
	}
	posSzFloat, _ := strconv.ParseFloat(posSize, 32)
	exitSz := (funcArgs["posPercToClose"].(float64) / 100) * posSzFloat

	return exitSz, nil
}

// OpenTradeSaga T-1
func cancelCalcCloseSize(allArgs ...interface{}) (interface{}, error) {
	fmt.Println("SAGA: Running cancelCalcCloseSize")
	// nothing to cancel
	return nil, nil
}

// OpenTradeSaga T2
func submitExitOrder(allArgs ...interface{}) (interface{}, error) {
	fmt.Println("SAGA: Running submitExitOrder")
	// XADD submitExitOrderIntent

	transactionArgs := allArgs[0].(map[string]interface{})

	msgs := []string{}
	msgs = append(msgs, "Action")
	msgs = append(msgs, "SubmitExitOrderIntent")
	msgs = append(msgs, "Symbol")
	msgs = append(msgs, "BTCUSDT")
	msgs = append(msgs, "Side")
	msgs = append(msgs, "SELL")
	msgs = append(msgs, "Quantity")
	msgs = append(msgs, "1000")
	msgs = append(msgs, "Price")
	msgs = append(msgs, "69")
	msgs = append(msgs, "Timestamp")
	msgs = append(msgs, time.Now().Format("2006-01-02_15:04:05_-0700"))
	msngr.AddToStream(transactionArgs["botStream"].(string), msgs)

	//listen for msg resp
	listenArgs := make(map[string]string)
	listenArgs["streamName"] = transactionArgs["botStream"].(string)
	listenArgs["groupName"] = svcConsumerGroupName
	listenArgs["consumerName"] = redisConsumerID
	listenArgs["start"] = ">"
	listenArgs["count"] = "1"

	//listen for consec responses
	return msngr.ListenConsecResponses(transactionArgs, "strat-svc submitExitOrder ListenConsecResponses", func(i int, v string, m redis.XMessage, isHeaderMatch bool) {
		fmt.Printf("Read consec header at index %v val: %s, IsMatch = %v (%s)\n", i, v, isHeaderMatch, m.ID)
	})

	// order-svc:
	//  exitOrderSubmitted, exitOrderFilled
	//  exitOrderFailed
	//  exitOrderSubmitted, exitOrderFilled
}

// OpenTradeSaga T-2
func cancelSubmitExitOrder(allArgs ...interface{}) (interface{}, error) {
	fmt.Println("SAGA: Running cancelSubmitExitOrder")

	// XADD cancelExitOrderIntent {timestamp}

	// order-svc:
	//  exitOrderCancelled
	return nil, nil
}

// edit SL/TP
var EditTrade saga.Saga

// OpenTradeSaga T1
func submitModifyPos(args map[string]interface{}) (interface{}, error) {
	// XADD submitModifyPosIntent {timestamp}

	// order-svc:
	//  modifyPosSubmitted, modifyPosSuccessful
	//  modifyPosFailed
	return nil, nil
}

// OpenTradeSaga T-1
func cancelModifyPos(args map[string]interface{}) (interface{}, error) {
	// modify back
	return nil, nil
}
