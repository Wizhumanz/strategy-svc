package main

import (
	"context"
	"log"
	"strconv"
	"time"

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

// TODO: eventually replaced by analytics svc

func saveMsg(streamMsg []string) {
	//build string to store
	var data string
	for _, r := range streamMsg {
		data = data + r + ","
	}

	ctx := context.Background()
	clientAdd, err := datastore.NewClient(ctx, googleProjectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	kind := "TradeAction"
	newBotKey := datastore.IncompleteKey(kind, nil)
	actionName := msngr.FindInStreamMsg(streamMsg, "HEADER")
	aggrID, _ := strconv.Atoi(msngr.FindInStreamMsg(streamMsg, "AGGR_ID"))
	sz, _ := strconv.ParseFloat(msngr.FindInStreamMsg(streamMsg, "SIZE"), 32)

	newAction := TradeAction{
		Action:      actionName,
		AggregateID: aggrID,
		BotID:       "", //TODO: fill later
		OrderType:   0,
		Size:        sz,
		Timestamp:   time.Now().Format("2006-01-02_15:04:05_-0700"),
		UserID:      "", //TODO: fill later
	}

	if _, err := clientAdd.Put(ctx, newBotKey, &newAction); err != nil {
		log.Fatalf("Failed to save TradeAction: %v", err)
	}
}
