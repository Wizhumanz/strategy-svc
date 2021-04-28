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
func calcPosSize(args map[string]interface{}) (interface{}, error) {
	fmt.Println("Running calcPosSize")
	return 69.69, nil
}

// OpenTradeSaga T-1
func cancelCalcPosSize(args map[string]interface{}) (interface{}, error) {
	fmt.Println("Running cancelCalcPosSize")
	// nothing to cancel
	return nil, nil
}

// OpenTradeSaga T2
func checkModel(args map[string]interface{}) (interface{}, error) {
	fmt.Println("CMD: Consulting ML model to decide if should take trade")
	//response: trade OK
	return nil, nil
}

// OpenTradeSaga T-2
func cancelCheckModel(args map[string]interface{}) (interface{}, error) {
	fmt.Println("Running cancelCheckModel")
	//nothing to compensate
	return nil, nil
}

// OpenTradeSaga T3
func submitEntryOrder(args map[string]interface{}) (interface{}, error) {
	fmt.Println("Running submitEntryOrder")

	// XADD submitEntryOrderIntent
	msgs := []string{}
	msgs = append(msgs, "Action")
	msgs = append(msgs, "SubmitEntryOrderIntent")
	msgs = append(msgs, "Size")
	msgs = append(msgs, args["0"].(string))
	msgs = append(msgs, "Timestamp")
	msgs = append(msgs, time.Now().Format("2006-01-02_15:04:05_-0700"))
	msngr.AddToStream(args["tradeStream"].(string), msgs)

	//listen for first resp from order-svc with CONSEC_RESP field
	consecRespHeaders := []string{}
	consecRespListenArgs := make(map[string]string)
	consecRespListenArgs["streamName"] = args["tradeStream"].(string)
	consecRespListenArgs["groupName"] = args["consumerGroup"].(string)
	consecRespListenArgs["consumerName"] = args["consumerID"].(string)
	consecRespListenArgs["start"] = ">"
	consecRespListenArgs["count"] = "1"
	var interConsecRespHeaders interface{}
	for {
		if len(consecRespHeaders) > 0 {
			break
		}

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

	//listen for consecutive responses with headers
	for i, hd := range consecRespHeaders {
		consecRespReadHandlers := []msngr.CommandHandler{
			{
				Command: "CONSEC_RESP",
				HandlerMatches: []msngr.HandlerMatch{
					{
						Matcher: func(fieldVal string) bool {
							return fieldVal != ""
						},
						Handler: func(msg redis.XMessage, output *interface{}) {
							//check msg for consec header
							consecHeader := msngr.FilterMsgVals(msg, func(key, val string) bool {
								return key == "CONSEC_HEADER"
							})
							if consecHeader == hd {
								fmt.Printf("Received consec header %v of %v: %s \n", i, len(consecRespHeaders), hd)
							}
						},
					},
				},
			},
		}

		msngr.ReadAndParse(msngr.ReadStream, msngr.ParseStream, consecRespListenArgs, consecRespReadHandlers)
	}

	// order-svc:
	//  entryOrderSubmitted, entryOrderFilled
	//  entryOrderFailed
	//  entryOrderSubmitted, entryOrderFilled, SLExitedTrade/TPExitedTrade
	return nil, nil
}

// OpenTradeSaga T-3
func cancelSubmitEntryOrder(args map[string]interface{}) (interface{}, error) {
	fmt.Println("Running cancelSubmitEntryOrder")

	// XADD cancelEntryOrderIntent {timestamp}

	// order-svc:
	//  entryOrderCancelled
	return nil, nil
}

// stop loss and take profit (maybe partial exits), and full exit
var ExitTradeSaga saga.Saga

// OpenTradeSaga T1
func calcCloseSize(args map[string]interface{}) (interface{}, error) {
	return 420.42, nil
}

// OpenTradeSaga T-1
func cancelCalcCloseSize(args map[string]interface{}) (interface{}, error) {
	// nothing to cancel
	return nil, nil
}

// OpenTradeSaga T2
func submitExitOrder(args map[string]interface{}) (interface{}, error) {
	// XADD submitExitOrderIntent {timestamp}

	// order-svc:
	//  exitOrderSubmitted, exitOrderFilled
	//  exitOrderFailed
	//  exitOrderSubmitted, exitOrderFilled
	return nil, nil
}

// OpenTradeSaga T-2
func cancelSubmitExitOrder(args map[string]interface{}) (interface{}, error) {
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
