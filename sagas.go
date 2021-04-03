package main

import "fmt"

var OpenLongSaga Saga

// OpenLongSaga T1
func submitTradeIntent() {
	fmt.Println("CMD: Submitting trade intent")
}

// OpenLongSaga T-1
func cancelSubmitTradeIntent() {
	fmt.Println("Trade intent cancelled")
}

// OpenLongSaga T2
func checkModel() {
	fmt.Println("CMD: Consulting ML model to decide if should take trade")
	//response: trade OK
}

// OpenLongSaga T-2
func cancelCheckModel() {
	//nothing to compensate
}

// OpenLongSaga T3
func submitEntryOrder() {
	fmt.Println("CMD: Submitting entry order")
	//response: entryOrderSubmitted, entryOrderFilled OR entryOrderSubmitted, entryOrderFailed OR entryOrderSubmitted, SLExitedTrade/TPExitedTrade
}

// OpenLongSaga T-3
func cancelSubmitEntryOrder() {
	//TODO: how to cancel entry order submissions?
}

// OpenLongSaga T4
func submitExitOrder() {
	fmt.Println("CMD: Exiting trade")
	//response: exitOrderSubmitted, exitOrderFilled OR exitOrderSubmitted, exitOrderFailed
}

// OpenLongSaga T-4
func cancelSubmitExitOrder() {
	//TODO: how to cancel exit order submissions?
}
