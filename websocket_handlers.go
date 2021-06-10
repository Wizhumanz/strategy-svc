package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

func readIncomingWsMsg(conn *websocket.Conn) {
	for {
		// messageType, p, err := conn.ReadMessage()
		_, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Println(string(p))
		// fmt.Printf("Msg type: %v\n", messageType)
	}
}

func wsConnectHandler(w http.ResponseWriter, r *http.Request) {
	// setupCORS(&w, r)
	// if (*r).Method == "OPTIONS" {
	// 	return
	// }

	ws, _ := upgrader.Upgrade(w, r, nil)

	//save connection globally
	wsConnections[mux.Vars(r)["id"]] = ws

	err := ws.WriteMessage(1, []byte("Yonkers motherfucker"))
	if err != nil {
		log.Println(err)
	}

	go readIncomingWsMsg(ws)
}

func wsChartmasterConnectHandler(w http.ResponseWriter, r *http.Request) {
	// setupCORS(&w, r)
	// if (*r).Method == "OPTIONS" {
	// 	return
	// }

	ws, _ := upgrader.Upgrade(w, r, nil)

	//save connection globally
	wsConnectionsChartmaster[mux.Vars(r)["id"]] = ws

	err := ws.WriteMessage(1, []byte("Yonkers Chartmaster"))
	if err != nil {
		log.Println(err)
	}

	go readIncomingWsMsg(ws)
}
