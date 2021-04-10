package main

import (
	"fmt"
	"strings"

	"gitlab.com/myikaco/msngr"
	"golang.org/x/crypto/bcrypt"
)

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
		last, streamMsgs := msngr.ListenStream(listenStreamName, lastRespID)
		lastRespID = last

		//parse response
		for _, strMsg := range streamMsgs {
			for _, m := range strMsg.MsgVals {
				msgs := []string{}
				msgs = append(msgs, "MSG")
				msgs = append(msgs, "hey there")
				msgs = append(msgs, "Order Size")
				msgs = append(msgs, "100x long bitch")
				msgs = append(msgs, "END")
				msgs = append(msgs, "END")

				switch m {
				case "ENTER":
					//find new trade stream name
					var newTradeStrName string
					for _, im := range strMsg.MsgVals {
						if strings.Contains(im, ":") {
							newTradeStrName = im
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
