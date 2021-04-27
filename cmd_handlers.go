package main

import (
	"fmt"

	"github.com/go-redis/redis/v8"
	"gitlab.com/myikaco/msngr"
)

func CmdEnterHandler(msg redis.XMessage, output *interface{}) {
	//find new trade stream name
	newTradeStrName := msngr.FilterMsgVals(msg, func(k, v string) bool {
		return (k == "TradeStreamName" && v != "")
	})

	//start OpenTradeSaga (triggers other svcs)
	OpenTradeSaga.Execute(newTradeStrName, svcConsumerGroupName, redisConsumerID)

	if newTradeStrName == "" {
		fmt.Println("\n" + colorRed + "New trade stream name empty!" + colorReset)
	}
}

func CmdExitHandler(msg redis.XMessage, output *interface{}) {
	fmt.Printf("EXIT cmd received for message %s", msg)
}

func CmdSLHandler(msg redis.XMessage, output *interface{}) {
	fmt.Printf("EXIT cmd received for message %s", msg)
}

func CmdTPHandler(msg redis.XMessage, output *interface{}) {
	fmt.Printf("EXIT cmd received for message %s", msg)
}

func ConsecRespAnyHandler(msg redis.XMessage, output *interface{}) {
	fmt.Printf("Inside consec resp handler for message %s and output %v", msg, &output)
}
