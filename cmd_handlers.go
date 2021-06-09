package main

import (
	"fmt"

	"github.com/go-redis/redis/v8"
	"gitlab.com/myikaco/msngr"
)

// helpers //

func handlerPrep(msg redis.XMessage) string {
	//find new trade stream name
	newTradeStrName := msngr.FilterMsgVals(msg, func(k, v string) bool {
		return (k == "BotStreamName" && v != "")
	})

	//create new consumer group for strategy-svc
	_, err := msngr.CreateNewConsumerGroup(newTradeStrName, svcConsumerGroupName, "0")
	if err != nil {
		fmt.Printf(colorRed+"%s Redis consumer group - %v\n"+colorReset, svcConsumerGroupName, err.Error())
	}

	return newTradeStrName
}

// handlers //

func StatusActivateHandler(msg redis.XMessage, output *interface{}) {
	newBotStreamName := handlerPrep(msg)
	if newBotStreamName == "" {
		fmt.Println("\n" + colorRed + "New bot stream name empty!" + colorReset)
		return
	}

	//listen on new bot stream
	go msngr.StreamListenLoop(newBotStreamName, ">", svcConsumerGroupName, redisConsumerID, "1", "0", botStreamCmdHandlers)

	msngr.AcknowledgeMsg(newBotStreamName, svcConsumerGroupName, redisConsumerID, msg.ID)
}

func CmdEnterHandler(msg redis.XMessage, output *interface{}) {
	newBotStreamName := handlerPrep(msg)
	if newBotStreamName == "" {
		fmt.Println("\n" + colorRed + "New bot stream name empty!" + colorReset)
		return
	}
	//start OpenTradeSaga (triggers other svcs)
	OpenTradeSaga.Execute(newBotStreamName, svcConsumerGroupName, redisConsumerID)
	fmt.Println(colorGreen + "\nSaga complete! " + newBotStreamName + colorReset)
}

func CmdExitHandler(msg redis.XMessage, output *interface{}) {
	newTradeStrName := handlerPrep(msg)
	if newTradeStrName == "" {
		fmt.Println("\n" + colorRed + "New trade stream name empty!" + colorReset)
		return
	}
	ExitTradeSaga.Execute(newTradeStrName, svcConsumerGroupName, redisConsumerID)
	fmt.Println(colorGreen + "\nSaga complete! " + newTradeStrName + colorReset)
}

func CmdSLHandler(msg redis.XMessage, output *interface{}) {
	fmt.Printf("EXIT cmd received for message %s", msg)
}

func CmdTPHandler(msg redis.XMessage, output *interface{}) {
	fmt.Printf("EXIT cmd received for message %s", msg)
}
