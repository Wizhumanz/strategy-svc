package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"gitlab.com/myikaco/msngr"
)

func initRedis() {
	// default to dev redis instance
	if redisHost == "" {
		redisHost = "127.0.0.1"
	}
	if redisPort == "" {
		redisPort = "6379"
	}
	fmt.Println(redisConsumerID + " Redis conn " + redisHost + ":" + redisPort)
	rdb = redis.NewClient(&redis.Options{
		Addr: redisHost + ":" + redisPort,
	})
}

func parseStream(stream []redis.XStream) {
	//parse response
	if len(stream) > 0 {
		for _, strMsg := range stream {
			for _, m := range strMsg.Messages {
				msgs := []string{}
				msgs = append(msgs, "MSG")
				msgs = append(msgs, "hey there")
				msgs = append(msgs, "Order Size")
				msgs = append(msgs, "100x long bitch")

				switch m.Values["CMD"] {
				case "ENTER":
					//TODO: start OpenTradeSaga

					//find new trade stream name
					var newTradeStrName string
					for _, fm := range strMsg.Messages {
						str := fm.Values["CMD"].(string)
						if strings.Contains(str, ":") {
							newTradeStrName = str
						}
					}
					//trigger other services
					fmt.Println("Adding to stream " + newTradeStrName)
					msngr.AddToStream(newTradeStrName, msgs)
				case "EXIT":
					fmt.Println("EXIT cmd received")
				case "SL":
					fmt.Println("SL cmd received")
				case "TP":
					fmt.Println("TP cmd received")
				}
			}
		}
	}
}

// readAndParse takes a stream reader function <readFunc> and stream message parser function <parserFunc>.
// It runs <readFunc> to get new stream messages and passes the result to <parserFunc> for processing.
// It returns a string which is either the lastID of the latest message read, or a message "OK" on successful claiming of a pending consumer group message.
func readAndParse(
	readFunc func(map[string]string) (interface{}, interface{}, error),
	parserFunc func([]redis.XStream),
	args map[string]string) (string, error) {
	var ret string

	a, b, error := readFunc(args)
	if lastID, ok := a.(string); ok {
		parserFunc(b.([]redis.XStream))
		return lastID, error
	}

	return fmt.Sprint(ret), nil
}

func autoClaimMsgsLoop(newTradeStream, consGroup, cons, minIdle, startID, count string) {
	args := make(map[string]string)
	args["streamName"] = newTradeStream
	args["groupName"] = consGroup
	args["consumerName"] = cons
	args["start"] = startID
	args["count"] = count
	args["minIdleTime"] = minIdle

	for {
		fmt.Println("\n Autoclaim old pending msgs...")
		msg, err := readAndParse(msngr.AutoClaimPendingMsgs, parseStream, args)
		if err != nil {
			fmt.Printf("%s \nSleeping 5 secs before retry", err.Error())
			time.Sleep(5000 * time.Millisecond)
		} else {
			fmt.Printf("Auto claim old msgs response: %v \nWaiting 10 secs before next listen...", msg)
			time.Sleep(10000 * time.Millisecond)
		}
	}
}

func streamListenLoop(listenStreamName, lastRespID, consumerGroup, consumerID, count string) {
	args := make(map[string]string)
	args["streamName"] = listenStreamName
	args["groupName"] = consumerGroup
	args["consumerName"] = consumerID
	args["start"] = lastRespID
	args["count"] = count

	for {
		fmt.Printf(colorRed+"\n %v listening on new trade stream %v...\n"+colorReset, consumerID, newTradeCmdStream)
		newLastMsgID, err := readAndParse(msngr.AutoClaimPendingMsgs, parseStream, args)
		if err != nil {
			fmt.Printf("%s \nSleeping 5 secs before retry", err.Error())
			time.Sleep(5000 * time.Millisecond)
		} else {
			args["start"] = newLastMsgID
			saveErr := msngr.SaveLastID(lastIDSaveKey, newLastMsgID)
			if saveErr != nil {
				fmt.Println(saveErr.Error())
			}
		}
	}
}
