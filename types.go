package main

import (
	"fmt"
	"reflect"
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
