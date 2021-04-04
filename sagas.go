package main

import "fmt"

var OpenLongSaga Saga

// OpenLongSaga T1
func checkModel() {
	fmt.Println("CMD: Consulting ML model to decide if should take trade")
	//response: trade OK
}

// OpenLongSaga T-1
func cancelCheckModel() {
	//nothing to compensate
}

// OpenLongSaga T2
func submitEntryOrder() {
	fmt.Println("CMD: Submitting entry order")
	//response: entryOrderSubmitted, entryOrderFilled OR entryOrderSubmitted, entryOrderFailed OR entryOrderSubmitted, entryOrderFilled, SLExitedTrade/TPExitedTrade
}

// OpenLongSaga T-2
func cancelSubmitEntryOrder() {
	//TODO: how to cancel entry order submissions?
}

// OpenLongSaga T3
func submitExitOrder() {
	fmt.Println("CMD: Exiting trade")
	//response: exitOrderSubmitted, exitOrderFilled OR exitOrderSubmitted, exitOrderFailed
}

// OpenLongSaga T-3
func cancelSubmitExitOrder() {
	//TODO: how to cancel exit order submissions?
}
