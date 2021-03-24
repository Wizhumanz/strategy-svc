package main

import (
	"context"
	// "encoding/base64"
	"encoding/json"
	"fmt"

	// "io"
	"log"
	"net/http"

	// "net/url"
	"os"
	"reflect"

	// "strings"
	"time"

	"cloud.google.com/go/datastore"
	// "cloud.google.com/go/storage"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
	// "google.golang.org/api/iterator"
)

// data types

type SagaStep struct {
	Transaction             func()
	CompensatingTransaction func()
}

type Saga struct {
	//iterate through this slice to execute saga
	//	each step runs stream listen loop to wait for new OK response,
	//		if response OK, break listen loop and proceeds with next step
	//			first OK response for each step from other svc includes message headers to listen for in next consecutive steps by the same svc
	//		if response FAIL, break all loops and start loop for compensating transactions
	Steps []SagaStep
}

var OpenLongSaga Saga

// OpenLongSaga T1
func submitTradeIntent() {
	fmt.Println("Submitting trade intent")
}

// OpenLongSaga T-1
func cancelSubmitTradeIntent() {
	fmt.Println("Trade intent cancelled")
}

// OpenLongSaga T2
func checkModel() {
	fmt.Println("Consulting ML model to decide if should take trade")
	//response: trade OK
}

// OpenLongSaga T-2
func cancelCheckModel() {
	//nothing to compensate
}

// OpenLongSaga T3
func submitEntryOrder() {
	fmt.Println("Submitting entry order")
	//response: entryOrderSubmitted, entryOrderFilled OR entryOrderSubmitted, entryOrderFailed OR entryOrderSubmitted, SLExitedTrade/TPExitedTrade
}

// OpenLongSaga T-3
func cancelSubmitEntryOrder() {
	//TODO: how to cancel entry order submissions?
}

// OpenLongSaga T4
func submitExitOrder() {
	fmt.Println("Exiting trade")
	//response: exitOrderSubmitted, exitOrderFilled OR exitOrderSubmitted, exitOrderFailed
}

// OpenLongSaga T-4
func cancelSubmitExitOrder() {
	//TODO: how to cancel exit order submissions?
}

// API types

type jsonResponse struct {
	Msg  string `json:"message"`
	Body string `json:"body"`
}

type webHookResponse struct {
	Msg  string `json:"message"`
	Size string `json:"size"`
}

//for unmarshalling JSON to bools
type JSONBool bool

func (bit *JSONBool) UnmarshalJSON(b []byte) error {
	txt := string(b)
	*bit = JSONBool(txt == "1" || txt == "true")
	return nil
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type User struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	AccountType string `json:"type"`
	Password    string `json:"password"`
}

func (l User) String() string {
	r := ""
	v := reflect.ValueOf(l)
	typeOfL := v.Type()

	for i := 0; i < v.NumField(); i++ {
		r = r + fmt.Sprintf("%s: %v, ", typeOfL.Field(i).Name, v.Field(i).Interface())
	}
	return r
}

var googleProjectID = "myika-anastasia"
var redisHost = os.Getenv("REDISHOST")
var redisPort = os.Getenv("REDISPORT")
var redisAddr = fmt.Sprintf("%s:%s", redisHost, redisPort)
var rdb *redis.Client

// helper funcs

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

// route handlers

func indexHandler(w http.ResponseWriter, r *http.Request) {
	var data jsonResponse
	w.Header().Set("Content-Type", "application/json")
	data = jsonResponse{Msg: "Strategy SVC Anastasia", Body: "Ready"}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
	// w.Write([]byte(`{"msg": "привет сука"}`))
}

func tvWebhookHandler(w http.ResponseWriter, r *http.Request) {
	//decode/unmarshall the body
	//two properties: "msg", "size"
	var webHookRes webHookResponse
	err := json.NewDecoder(r.Body).Decode(&webHookRes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Println(webHookRes.Msg)
	fmt.Println(webHookRes.Size)
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

func main() {
	//init sagas
	OpenLongSaga = Saga{
		Steps: []SagaStep{
			{Transaction: submitTradeIntent, CompensatingTransaction: cancelSubmitTradeIntent},
			{Transaction: checkModel, CompensatingTransaction: cancelCheckModel},
			{Transaction: submitEntryOrder, CompensatingTransaction: cancelSubmitEntryOrder},
			{Transaction: submitExitOrder, CompensatingTransaction: cancelSubmitExitOrder},
		},
	}

	// initRedis()
	// go startStream()

	router := mux.NewRouter().StrictSlash(true)
	router.Methods("GET").Path("/").HandlerFunc(indexHandler)
	router.Methods("POST").Path("/tv-hook").HandlerFunc(tvWebhookHandler)
	// router.Methods("POST").Path("/user").HandlerFunc(createNewUserHandler)
	// router.Methods("POST").Path("/owner").HandlerFunc(createNewUserHandler)
	// router.Methods("GET").Path("/listings").HandlerFunc(getAllListingsHandler)
	// router.Methods("POST").Path("/listing").HandlerFunc(createNewListingHandler)
	// router.Methods("PUT").Path("/listing/{id}").HandlerFunc(updateListingHandler)

	port := os.Getenv("PORT")
	fmt.Println("strategy-svc listening on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
