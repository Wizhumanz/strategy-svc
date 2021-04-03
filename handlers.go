package main

import (
	"encoding/json"
	"fmt"
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

func tvWebhookHandler(w http.ResponseWriter, r *http.Request) {
	//decode/unmarshall the body
	//two properties: "msg", "size"
	var webHookRes webHookResponse
	err := json.NewDecoder(r.Body).Decode(&webHookRes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Println(webHookRes.Msg)
	fmt.Println(webHookRes.Size)
}
