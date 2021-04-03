package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/go-redis/redis/v8"
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

//returns last ID and response from XREAD
func listenStream(streamName string, lastID string) (string, []string) {
	var ret []string
	var retLastID string = ""
	var ctx = context.Background()

	//keep XREAD until got new response on passed stream
	for {
		streams, err := rdb.XRead(ctx, &redis.XReadArgs{
			Streams: []string{streamName, lastID},
			Block:   500,
		}).Result()

		if err != nil {
			fmt.Println("XREAD error -- ", err.Error())
			fmt.Println("Sleeping 5s before retry...")
			time.Sleep(5000 * time.Millisecond)
		} else {
			retLastID = streams[len(streams)-1].Messages[0].ID
			//for each message, add each key-value pair to return array
			for _, stream := range streams[0].Messages {
				for key, val := range stream.Values {
					if val != "" {
						ret = append(ret, fmt.Sprintf("Msg %s: %s = %s", stream.ID, key, val)) //use a map instead
					}
				}
			}
			//stop listening once got some result
			break
		}
	}
	return retLastID, ret
}

//TODO: modify this function to take a lambda that executes on every key/value in msg
func findInStreamMsg(streamResp []string, key string) string {
	var data string
	readKey := false
	for _, r := range streamResp {
		if r == key {
			readKey = true
		} else if readKey {
			readKey = false
			data = r
		}
	}
	return data
}

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
	actionName := findInStreamMsg(streamMsg, "HEADER")
	aggrID, _ := strconv.Atoi(findInStreamMsg(streamMsg, "AGGR_ID"))
	sz, _ := strconv.ParseFloat(findInStreamMsg(streamMsg, "SIZE"), 32)

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

func initRedis() {
	if redisHost == "" {
		redisHost = "127.0.0.1"
		fmt.Println("Env var nil, using redis dev address -- " + redisHost)
	}
	if redisPort == "" {
		redisPort = "6379"
		fmt.Println("Env var nil, using redis dev port -- " + redisPort)
	}
	fmt.Println("Connected to Redis on " + redisHost + ":" + redisPort)
	rdb = redis.NewClient(&redis.Options{
		Addr: redisHost + ":" + redisPort,
	})
}

func startStream() {
	tempStreamName := "userEvents"
	var ctxStrat = context.Background()

	for {
		newID, err := rdb.XAdd(ctxStrat, &redis.XAddArgs{
			Stream: tempStreamName,
			Values: []string{
				"newEvent",
				time.Now().Local().String(),
			},
		}).Result()
		if err != nil {
			log.Fatal("XADD error -- ", err.Error())
		}

		l, xlenErr := rdb.Do(ctxStrat, "XLEN", tempStreamName).Result()
		if xlenErr != nil {
			log.Fatal("XLEN error -- ", xlenErr.Error())
		}

		if newID != "" {
			fmt.Print("Added to stream " + newID + " / len = ")
			fmt.Println(l)
		}

		time.Sleep(1500 * time.Millisecond)

		if l.(int64) > 10 {
			fmt.Println("Resetting DB with flushall...")
			rdb.Do(ctxStrat, "flushall").Result()
		}
	}
}
