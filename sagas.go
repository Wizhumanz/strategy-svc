package main

import (
	"fmt"
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
			Transaction:             checkModel,
			CompensatingTransaction: cancelCheckModel,
		},
		{
			Transaction:             submitEntryOrder,
			CompensatingTransaction: cancelSubmitEntryOrder,
		},
	},
}

// OpenTradeSaga T1
func calcPosSize(args map[string]interface{}) (interface{}, error) {
	fmt.Println("SAGA: Running calcPosSize")
	return "69.69", nil
}

// OpenTradeSaga T-1
func cancelCalcPosSize(args map[string]interface{}) (interface{}, error) {
	fmt.Println("SAGA: Running cancelCalcPosSize")
	// nothing to cancel
	return nil, nil
}

// OpenTradeSaga T2
func checkModel(args map[string]interface{}) (interface{}, error) {
	fmt.Println("SAGA: Consulting ML model to decide if should take trade")
	//response: trade OK
	return nil, nil
}

// OpenTradeSaga T-2
func cancelCheckModel(args map[string]interface{}) (interface{}, error) {
	fmt.Println("SAGA: Running cancelCheckModel")
	//nothing to compensate
	return nil, nil
}

// OpenTradeSaga T3
func submitEntryOrder(args map[string]interface{}) (interface{}, error) {
	fmt.Println("SAGA: Running submitEntryOrder")

	// XADD submitEntryOrderIntent
	msgs := []string{}
	msgs = append(msgs, "Action")
	msgs = append(msgs, "SubmitEntryOrderIntent")
	msgs = append(msgs, "Size")
	msgs = append(msgs, args["0"].(string))
	msgs = append(msgs, "Timestamp")
	msgs = append(msgs, time.Now().Format("2006-01-02_15:04:05_-0700"))
	msngr.AddToStream(args["tradeStream"].(string), msgs)

	//listen for consec responses
	msngr.ListenConsecResponses(args, func(i int, v string, m redis.XMessage, isHeaderMatch bool) {
		fmt.Printf("Read consec header at index %v val: %s, IsMatch = %v (%s)", i, v, isHeaderMatch, m.ID)
	})

	// order-svc:
	//  entryOrderSubmitted, entryOrderFilled
	//  entryOrderFailed
	//  entryOrderSubmitted, entryOrderFilled, SLExitedTrade/TPExitedTrade
	return nil, nil
}

// OpenTradeSaga T-3
func cancelSubmitEntryOrder(args map[string]interface{}) (interface{}, error) {
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
func calcCloseSize(args map[string]interface{}) (interface{}, error) {
	fmt.Println("SAGA: Running calcCloseSize")
	return 420.42, nil
}

// OpenTradeSaga T-1
func cancelCalcCloseSize(args map[string]interface{}) (interface{}, error) {
	fmt.Println("SAGA: Running cancelCalcCloseSize")
	// nothing to cancel
	return nil, nil
}

// OpenTradeSaga T2
func submitExitOrder(args map[string]interface{}) (interface{}, error) {
	fmt.Println("SAGA: Running submitExitOrder")

	// XADD submitExitOrderIntent
	msgs := []string{}
	msgs = append(msgs, "Action")
	msgs = append(msgs, "SubmitExitOrderIntent")
	msgs = append(msgs, "Size")
	msgs = append(msgs, args["0"].(string))
	msgs = append(msgs, "Timestamp")
	msgs = append(msgs, time.Now().Format("2006-01-02_15:04:05_-0700"))
	msngr.AddToStream(args["tradeStream"].(string), msgs)

	//listen for consec responses
	return msngr.ListenConsecResponses(args, func(i int, v string, m redis.XMessage, isHeaderMatch bool) {
		fmt.Printf("Read consec header at index %v val: %s, IsMatch = %v (%s)", i, v, isHeaderMatch, m.ID)
	})

	// order-svc:
	//  exitOrderSubmitted, exitOrderFilled
	//  exitOrderFailed
	//  exitOrderSubmitted, exitOrderFilled
}

// OpenTradeSaga T-2
func cancelSubmitExitOrder(args map[string]interface{}) (interface{}, error) {
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
