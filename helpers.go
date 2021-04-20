package main

import (
	"fmt"
	"strings"

	"github.com/go-redis/redis/v8"
	"gitlab.com/myikaco/msngr"
)

// func initRedis() {
// 	// default to dev redis instance
// 	if redisHost == "" {
// 		redisHost = "127.0.0.1"
// 		fmt.Println("Env var nil, using redis dev address -- " + redisHost)
// 	}
// 	if redisPort == "" {
// 		redisPort = "6379"
// 		fmt.Println("Env var nil, using redis dev port -- " + redisPort)
// 	}
// 	fmt.Println("Connecting to Redis on " + redisHost + ":" + redisPort)
// 	rdb = redis.NewClient(&redis.Options{
// 		Addr: redisHost + ":" + redisPort,
// 	})
// }

// readAndParse takes a stream reader function <readFunc> and stream message parser function <parserFunc>.
// It runs <readFunc> to get new stream messages and passes the result to <parserFunc> for processing.
// It returns a string which is either the lastID of the latest message read, or a message "OK" on successful claiming of a pending consumer group message.
func readAndParse(
	readFunc func(...string) (interface{}, interface{}),
	parserFunc func([]redis.XStream),
	newTradeStream, consGroup, cons, minIdle, startID, count string) string {
	var ret string
	if minIdle != "" {
		msg, _ := readFunc(newTradeStream, consGroup, cons, startID, count, minIdle)
		ret = msg.(string)
	} else {
		lastID, _ := readFunc(newTradeStream, consGroup, cons, startID, count)
		ret = lastID.(string)
	}

	return ret
}

//TODO: two funcs below use new readAndParse func
//TODO: refactor out parser func

func autoClaimMsgsLoop(newTradeStream, consGroup, cons, minIdle, startID, count string) {
	msngr.AutoClaimPendingMsgs(newTradeStream, consGroup, cons, startID, count, minIdle)
}

func streamListenLoop(listenStreamName, lastRespID, consumerGroup, consumerID, count string) {
	for {
		fmt.Println("\nListening...")
		last, streamMsgs := msngr.ReadStream(
			listenStreamName,
			consumerGroup,
			consumerID,
			lastRespID,
			count)

		//save last ID to only get new msgs later
		lastRespID = last
		saveErr := msngr.SaveLastID(lastIDSaveKey, last)
		if saveErr != nil {
			fmt.Println(saveErr.Error())
		}

		//parse response
		if len(streamMsgs) > 0 {
			for _, strMsg := range streamMsgs {
				for _, m := range strMsg.Messages {
					msgs := []string{}
					msgs = append(msgs, "MSG")
					msgs = append(msgs, "hey there")
					msgs = append(msgs, "Order Size")
					msgs = append(msgs, "100x long bitch")
					msgs = append(msgs, "END")
					msgs = append(msgs, "END")

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
}
