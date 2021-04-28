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

	//create new consumer group for strategy-svc
	_, err := msngr.CreateNewConsumerGroup(newTradeStrName, svcConsumerGroupName, "0")
	if err != nil {
		fmt.Printf(colorRed+"%s Redis consumer group - %v\n"+colorReset, svcConsumerGroupName, err.Error())
	}
	//start OpenTradeSaga (triggers other svcs)
	OpenTradeSaga.Execute(newTradeStrName, svcConsumerGroupName, redisConsumerID)
	fmt.Println(colorGreen + "Saga complete! " + newTradeStrName + colorReset)

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
