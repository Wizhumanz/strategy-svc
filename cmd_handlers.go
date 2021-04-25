package main

import (
	"fmt"

	"github.com/go-redis/redis/v8"
	"gitlab.com/myikaco/msngr"
)

func CmdEnterHandler(msg redis.XMessage) {
	//find new trade stream name
	newTradeStrName := msngr.FilterMsgVals(msg, func(k, v string) bool {
		return (k == "TradeStreamName" && v != "")
	})

	//start OpenTradeSaga

	if newTradeStrName == "" {
		fmt.Println("\n" + colorRed + "New trade stream name empty!" + colorReset)
	} else {
		//trigger other services
		msgs := []string{}
		msgs = append(msgs, "MSG")
		msgs = append(msgs, "hey there")
		msgs = append(msgs, "Order Size")
		msgs = append(msgs, "100x long bitch")
		msngr.AddToStream(newTradeStrName, msgs)
	}
}

func CmdExitHandler(msg redis.XMessage) {
	fmt.Printf("EXIT cmd received for message %s", msg)
}

func CmdSLHandler(msg redis.XMessage) {
	fmt.Printf("EXIT cmd received for message %s", msg)
}

func CmdTPHandler(msg redis.XMessage) {
	fmt.Printf("EXIT cmd received for message %s", msg)
}
