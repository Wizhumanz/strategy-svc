package main

import (
	"fmt"
	"strings"

	"gitlab.com/myikaco/msngr"
	"golang.org/x/crypto/bcrypt"
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

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 2)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func streamListenLoop(listenStreamName string, lastRespID string) {
	for {
		fmt.Println("\nListening...")
		last, streamMsgs := msngr.ReadStream(listenStreamName, lastRespID, "", "", 1)

		//save last ID to only get new msgs later
		lastRespID = last
		saveErr := msngr.SaveLastID(lastIDSaveKey, last)
		if saveErr != nil {
			fmt.Println(saveErr.Error())
		}

		//parse response
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
