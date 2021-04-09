package main

import (
	"fmt"

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
		fmt.Println("Listening...")
		last, streamMsg := msngr.ListenStream(listenStreamName, lastRespID)
		lastRespID = last

		//parse response
		for _, r := range streamMsg {
			fmt.Println(r)
			//TODO: switch statement with fancy new map return type
		}
	}
}
