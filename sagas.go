package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"gitlab.com/myikaco/msngr"
	"gitlab.com/myikaco/saga"
)

var OpenTradeSaga saga.Saga

// OpenTradeSaga T1
func calcPosSize(args ...interface{}) (interface{}, error) {
	fmt.Println("Running calcPosSize")
	return 69.69, nil
}

// OpenTradeSaga T-1
func cancelCalcPosSize(args ...interface{}) (interface{}, error) {
	fmt.Println("Running cancelCalcPosSize")
	// nothing to cancel
	return nil, nil
}

// OpenTradeSaga T2
func checkModel(args ...interface{}) (interface{}, error) {
	fmt.Println("CMD: Consulting ML model to decide if should take trade")
	//response: trade OK
	return nil, nil
}

// OpenTradeSaga T-2
func cancelCheckModel(args ...interface{}) (interface{}, error) {
	fmt.Println("Running cancelCheckModel")
	//nothing to compensate
	return nil, nil
}

// OpenTradeSaga T3
func submitEntryOrder(args ...interface{}) (interface{}, error) {
	fmt.Println("Running submitEntryOrder")

	// XADD submitEntryOrderIntent
	msgs := []string{}
	msgs = append(msgs, "Action")
	msgs = append(msgs, "SubmitEntryOrderIntent")
	msgs = append(msgs, "Timestamp")
	msgs = append(msgs, time.Now().Format("2006-01-02_15:04:05_-0700"))
	msngr.AddToStream(args[0].(string), msgs)

	//listen for first resp from order-svc with CONSEC_RESP field
	consecRespHeaders := []string{}
	var interConsecRespHeaders interface{}
	for {
		if len(consecRespHeaders) > 0 {
			break
		}

		consecRespListenArgs := make(map[string]string)
		consecRespListenArgs["streamName"] = args[0].(string)
		consecRespListenArgs["groupName"] = args[1].(string)
		consecRespListenArgs["consumerName"] = args[2].(string)
		consecRespListenArgs["start"] = ">"
		consecRespListenArgs["count"] = "1"

		consecRespReadHandlers := []msngr.CommandHandler{
			{
				Command: "CONSEC_RESP",
				HandlerMatches: []msngr.HandlerMatch{
					{
						Matcher: func(fieldVal string) bool {
							return fieldVal != ""
						},
						Handler: func(msg redis.XMessage, output *interface{}) {
							fmt.Printf("Inside consec resp handler for message %s and output %v", msg, &output)

							//process consec responses to &output arg
							interConsecRespHeaders = msngr.FilterMsgVals(msg, func(key, val string) bool {
								return key == "CONSEC_RESP"
							})
						},
					},
				},
			},
		}
		msngr.ReadAndParse(msngr.ReadStream, msngr.ParseStream, consecRespListenArgs, consecRespReadHandlers)
		//TODO: how to react if received the wrong message?

		//convert output interface{} to []string{}
		fmt.Println(interConsecRespHeaders)
		if interConsecRespHeaders != nil {
			if conv, ok := interConsecRespHeaders.(string); ok {
				consecRespHeaders = strings.Split(conv, ",")
			} else {
				return nil, fmt.Errorf("could not convert consecutive response header field with value %s", interConsecRespHeaders)
			}
		}
	}

	//TODO: fill out + test entire saga
	//TODO: fill other sagas

	//TODO: listen for consecutive responses with headers
	for _, hd := range consecRespHeaders {
		fmt.Println(hd)
	}

	// order-svc:
	//  entryOrderSubmitted, entryOrderFilled
	//  entryOrderFailed
	//  entryOrderSubmitted, entryOrderFilled, SLExitedTrade/TPExitedTrade
	return nil, nil
}

// OpenTradeSaga T-3
func cancelSubmitEntryOrder(args ...interface{}) (interface{}, error) {
	fmt.Println("Running cancelSubmitEntryOrder")

	// XADD cancelEntryOrderIntent {timestamp}

	// order-svc:
	//  entryOrderCancelled
	return nil, nil
}

// stop loss and take profit (maybe partial exits), and full exit
var ExitTradeSaga saga.Saga

// OpenTradeSaga T1
func calcCloseSize(args ...interface{}) (interface{}, error) {
	return 420.42, nil
}

// OpenTradeSaga T-1
func cancelCalcCloseSize(args ...interface{}) (interface{}, error) {
	// nothing to cancel
	return nil, nil
}

// OpenTradeSaga T2
func submitExitOrder(args ...interface{}) (interface{}, error) {
	// XADD submitExitOrderIntent {timestamp}

	// order-svc:
	//  exitOrderSubmitted, exitOrderFilled
	//  exitOrderFailed
	//  exitOrderSubmitted, exitOrderFilled
	return nil, nil
}

// OpenTradeSaga T-2
func cancelSubmitExitOrder(args ...interface{}) (interface{}, error) {
	// XADD cancelExitOrderIntent {timestamp}

	// order-svc:
	//  exitOrderCancelled
	return nil, nil
}

// edit SL/TP
var EditTrade saga.Saga

// OpenTradeSaga T1
func submitModifyPos(args ...interface{}) (interface{}, error) {
	// XADD submitModifyPosIntent {timestamp}

	// order-svc:
	//  modifyPosSubmitted, modifyPosSuccessful
	//  modifyPosFailed
	return nil, nil
}

// OpenTradeSaga T-1
func cancelModifyPos(args ...interface{}) (interface{}, error) {
	// modify back
	return nil, nil
}
