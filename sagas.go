package main

import (
	"fmt"

	"gitlab.com/myikaco/saga"
)

var OpenTradeSaga saga.Saga

// OpenTradeSaga T1
func calcPosSize() float64 {
	return 69.69
}

// OpenTradeSaga T-1
func cancelCalcPosSize() {
	// nothing to cancel
}

// OpenTradeSaga T2
func checkModel() {
	fmt.Println("CMD: Consulting ML model to decide if should take trade")
	//response: trade OK
}

// OpenTradeSaga T-2
func cancelCheckModel() {
	//nothing to compensate
}

// OpenTradeSaga T3
func submitEntryOrder() {
	// XADD submitEntryOrderIntent {timestamp}

	// order-svc:
	//  entryOrderSubmitted, entryOrderFilled
	//  entryOrderFailed
	//  entryOrderSubmitted, entryOrderFilled, SLExitedTrade/TPExitedTrade
}

// OpenTradeSaga T-3
func cancelSubmitEntryOrder() {
	// XADD cancelEntryOrderIntent {timestamp}

	// order-svc:
	//  entryOrderCancelled
}

// stop loss and take profit (maybe partial exits), and full exit
var ExitTradeSaga saga.Saga

// OpenTradeSaga T1
func calcCloseSize() float64 {
	return 420.42
}

// OpenTradeSaga T-1
func cancelCalcCloseSize() {
	// nothing to cancel
}

// OpenTradeSaga T2
func submitExitOrder() {
	// XADD submitExitOrderIntent {timestamp}

	// order-svc:
	//  exitOrderSubmitted, exitOrderFilled
	//  exitOrderFailed
	//  exitOrderSubmitted, exitOrderFilled
}

// OpenTradeSaga T-2
func cancelSubmitExitOrder() {
	// XADD cancelExitOrderIntent {timestamp}

	// order-svc:
	//  exitOrderCancelled
}

// edit SL/TP
var EditTrade saga.Saga

// OpenTradeSaga T1
func submitModifyPos() {
	// XADD submitModifyPosIntent {timestamp}

	// order-svc:
	//  modifyPosSubmitted, modifyPosSuccessful
	//  modifyPosFailed
}

// OpenTradeSaga T-1
func cancelModifyPos() {
	// modify back
}
