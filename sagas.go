package main

import (
	"fmt"

	"gitlab.com/myikaco/saga"
)

var OpenTradeSaga saga.Saga

// OpenTradeSaga T1
func calcPosSize(args ...interface{}) (interface{}, error) {
	return 69.69, nil
}

// OpenTradeSaga T-1
func cancelCalcPosSize(args ...interface{}) (interface{}, error) {
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
	//nothing to compensate
	return nil, nil
}

// OpenTradeSaga T3
func submitEntryOrder(args ...interface{}) (interface{}, error) {
	// XADD submitEntryOrderIntent {timestamp}

	// order-svc:
	//  entryOrderSubmitted, entryOrderFilled
	//  entryOrderFailed
	//  entryOrderSubmitted, entryOrderFilled, SLExitedTrade/TPExitedTrade
	return nil, nil
}

// OpenTradeSaga T-3
func cancelSubmitEntryOrder(args ...interface{}) (interface{}, error) {
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
