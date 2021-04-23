package main

import (
	"fmt"
	"strings"

	"github.com/go-redis/redis/v8"
	"gitlab.com/myikaco/msngr"
)

func CmdEnterHandler(msg redis.XMessage) {
	//TODO: start OpenTradeSaga

	//find new trade stream name
	//TODO: convert into generic map filter func
	var newTradeStrName string
	for key, val := range msg.Values {
		if key == "TradeStreamName" && val.(string) != "" {
			str := val.(string)
			if strings.Contains(str, ":") {
				newTradeStrName = str
			}
		}
	}

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
		fmt.Println("\nAdding to stream " + newTradeStrName)
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
