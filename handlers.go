package main

import (
	"encoding/json"
	"net/http"
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	var data jsonResponse
	w.Header().Set("Content-Type", "application/json")
	data = jsonResponse{Msg: "Strategy SVC Anastasia", Body: "Ready"}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
	// w.Write([]byte(`{"msg": "привет сука"}`))
}

//Process for executing trade actions
// 1. TV webhook calls api-gateway /webhook route
// 2. api-gateway adds new trade info (<userID>:<aggregateID>) into webhookTrades stream
// 3. strategy-svc executes saga for each msg in webhookTrades stream

func newTradeHandler(w http.ResponseWriter, r *http.Request) {
	//PASSED FROM CALLER: []{userID, trade aggregateID, botData} (run trade on each), signal data (long/short, SL/TP, etc)
}
