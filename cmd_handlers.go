package main

import (
	"encoding/json"
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
	go msngr.StreamListenLoop(newBotStreamName, "strat-svc StatusActivateHandler", ">", svcConsumerGroupName, redisConsumerID, "1", "0", botStreamCmdHandlers, stopListenCmdChecker)

	//start live strat execution loop
	botInfo := msngr.FilterMsgVals(msg, func(k, v string) bool {
		return (k == "Bot" && v != "")
	})
	var bot Bot
	json.Unmarshal([]byte(botInfo), &bot)
	// go executeLiveStrategy(bot, strat1)

	msngr.AcknowledgeMsg(newBotStreamName, svcConsumerGroupName, redisConsumerID, msg.ID)
}
