package main

import (
	"fmt"
	"reflect"
	"strings"
)

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

type TradeAction struct {
	Action      string
	AggregateID int
	UserID      string
	BotID       string
	OrderType   int
	Size        float64
	Timestamp   string
}

// sagas

type SagaStep struct {
	Transaction             func()
	CompensatingTransaction func()
}

type Saga struct {
	//iterate through this slice to execute saga
	//	each step runs stream listen loop to wait for new OK response,
	//		if response OK, break listen loop and proceed with next step
	//			first OK response for each step from other svc includes message headers to listen for in next consecutive steps by the same svc
	//		if response FAIL, break all loops and start loop for compensating transactions
	Steps []SagaStep
}

func (saga *Saga) UndoSaga(listenStreamName string, lastStep int, lastMsgID string) {
	lastRespID := lastMsgID //listen from stream start by default
	for i := len(saga.Steps) - 1; i >= 0; i-- {
		if i <= lastStep {
			//execute compensating transaction, listen for response
			step := saga.Steps[i]
			step.CompensatingTransaction()
			last, streamResponses := listenStream(listenStreamName, lastRespID)
			lastRespID = last

			//parse response
			for _, r := range streamResponses {
				fmt.Println(r)
			}
		}
	}
}

func (saga *Saga) Execute(listenStreamName string) {
	lastRespID := "0" //listen from stream start by default
	for i, step := range saga.Steps {
		undoSaga := false
		//execute saga step, listen for response
		step.Transaction()
		//TODO: put "listen and parse loop" below in external function
		last, streamMsg := listenStream(listenStreamName, lastRespID)
		lastRespID = last

		//save + parse response
		go saveMsg(streamMsg)
		for _, r := range streamMsg {
			fmt.Println(r)

			if r == "ERROR" {
				undoSaga = true
				break
			}
		}
		if !undoSaga {
			//check for consecutive response header
			consecMsgHeaders := strings.Split(findInStreamMsg(streamMsg, "CONSEC_MSGS"), ",")
			//listen for next consecutive messages (if any)
			//TODO: handle unexpected messages being received
			for _, msg := range consecMsgHeaders {
				l, consecResp := listenStream(listenStreamName, lastRespID)
				lastRespID = l
				for _, d := range consecResp {
					if d == msg {
						fmt.Printf("Consec msg with header %s successfully received!", msg)
					}
					fmt.Println(d)
				}
			}
		}

		//if failed at any point, begin compensating transaction loop
		if undoSaga {
			saga.UndoSaga(listenStreamName, i, lastRespID)
		}
	}
}
