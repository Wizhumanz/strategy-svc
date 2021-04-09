package main

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/datastore"
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

func authenticateUser(req loginReq) bool {
	// get user with email
	ctx := context.Background()
	client, err := datastore.NewClient(ctx, googleProjectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	var userWithEmail User
	query := datastore.NewQuery("User").
		Filter("Email =", req.Email)
	t := client.Run(ctx, query)
	_, error := t.Next(&userWithEmail)
	if error != nil {
		// Handle error.
	}

	// check password hash and return
	return CheckPasswordHash(req.Password, userWithEmail.Password)
}

func streamListenLoop(listenStreamName, lastRespID string) {
	for {
		last, streamMsg := msngr.ListenStream(listenStreamName, lastRespID)
		lastRespID = last

		//parse response
		for _, r := range streamMsg {
			fmt.Println(r)
		}
	}
}
