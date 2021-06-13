package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/datastore"
)

type LogData struct {
	Timestamp string `json:"Timestamp"`
	Log       string `json:"Log"`
	Location  string `json:"Location"`
	Repo      string `json:"Repo"`
}

func Log(data, location string) {
	//print to console if in dev mode
	if os.Getenv("PRINTLOGS") == "" {
		newLog := LogData{
			Timestamp: time.Now().UTC().Format(httpTimeFormat),
			Log:       data,
			Location:  location,
			Repo:      svcConsumerGroupName,
		}

		ctx := context.Background()
		var newKey *datastore.Key
		clientAdd, err := datastore.NewClient(ctx, googleProjectID)
		if err != nil {
			log.Fatalf("Failed to create client: %v", err)
		}
		kind := "Log"
		newKey = datastore.IncompleteKey(kind, nil)

		if _, err := clientAdd.Put(ctx, newKey, &newLog); err != nil {
			log.Fatalf("Failed to save LogData: %v", err)
		}
	} else {
		fmt.Printf("%v\n"+colorCyan+"%v\n"+colorReset, data, location)
	}
}
