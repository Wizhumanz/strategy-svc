package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"
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

var fileName string

func createJSONFile(botName, period string) {
	fileName = fmt.Sprintf("%v %v.json", botName, period)
	mydata := []byte(botName + " " + period + ":\n")

	// the WriteFile method returns an error if unsuccessful
	err := ioutil.WriteFile(fileName, mydata, 0777)
	// handle this error
	if err != nil {
		// print it out
		fmt.Println(err)
	}
}

func loggingInJSON(text string) string {
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		go Log(err.Error(),
			fmt.Sprintf("<%v> %v", line, file))
		return ""
	}
	defer f.Close()

	if _, err = f.WriteString(text + "\n"); err != nil {
		_, file, line, _ := runtime.Caller(0)
		go Log(err.Error(),
			fmt.Sprintf("<%v> %v", line, file))
		return ""
	}

	return text
}
