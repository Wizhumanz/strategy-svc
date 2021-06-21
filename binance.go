package main

import (
	"fmt"
	"runtime"
)

func getFuturesAccountBalance() []byte {
	// timeStamp := makeTimestamp()

	// secret := "BfqSCwpNCslkepaOO7dTejFRz5thaGiTUBX1p4fZp6sDPDuJrtmNt6Wse9hMpTOF"
	// data := fmt.Sprintf("timestamp=%d", timeStamp)

	// // Create a new HMAC by defining the hash type and the key (as byte array)
	// h := hmac.New(sha256.New, []byte(secret))

	// // Write Data to it
	// h.Write([]byte(data))

	// // Get result and encode as hexadecimal string
	// signature := hex.EncodeToString(h.Sum(nil))

	// req, _ := http.NewRequest("GET", fmt.Sprintf("https://fapi.binance.com/fapi/v2/balance?timestamp=%d&signature=%s", timeStamp, signature), nil)
	// req.Header.Set("Content-Type", "application/json")
	// req.Header.Add("X-MBX-APIKEY", "klGMQA5VZzL5dhi2DuR4agiYgVZaF8gxmQ0ZEuYkyfURRymazrIYtIBd2TtEheRp")
	// client := &http.Client{}

	// response, err := client.Do(req)
	// if err != nil {
	// 	log.Fatalf("An Error Occured %v", err)
	// 	return nil
	// } else {
	// 	body, _ := ioutil.ReadAll(response.Body)
	// 	fmt.Println(response.Body)
	// 	return body
	// }
	_, file, line, _ := runtime.Caller(0)
	go Log("getFuturesAccountBalance", fmt.Sprintf("<%v> %v", line, file))
	return nil
}

func changeMarginType() {
	// timeStamp := makeTimestamp()

	// secret := "BfqSCwpNCslkepaOO7dTejFRz5thaGiTUBX1p4fZp6sDPDuJrtmNt6Wse9hMpTOF"
	// data := fmt.Sprintf("symbol=BTCUSDT&marginType=ISOLATED&timestamp=%d", timeStamp)

	// // Create a new HMAC by defining the hash type and the key (as byte array)
	// h := hmac.New(sha256.New, []byte(secret))

	// // Write Data to it
	// h.Write([]byte(data))

	// // Get result and encode as hexadecimal string
	// signature := hex.EncodeToString(h.Sum(nil))

	// req, _ := http.NewRequest("POST", fmt.Sprintf("https://fapi.binance.com/fapi/v1/marginType?symbol=BTCUSDT&marginType=ISOLATED&timestamp=%d&signature=%s", timeStamp, signature), nil)
	// req.Header.Set("Content-Type", "application/json")
	// req.Header.Add("X-MBX-APIKEY", "klGMQA5VZzL5dhi2DuR4agiYgVZaF8gxmQ0ZEuYkyfURRymazrIYtIBd2TtEheRp")
	// client := &http.Client{}

	// response, err := client.Do(req)

	// if err != nil {
	// 	log.Fatalf("An Error Occured %v", err)
	// } else {
	// 	body, _ := ioutil.ReadAll(response.Body)
	// 	log.Println(string(body))
	// }
	_, file, line, _ := runtime.Caller(0)
	go Log("changeMarginType", fmt.Sprintf("<%v> %v", line, file))
}

func changeInitialLeverage() {
	// timeStamp := makeTimestamp()

	// secret := "BfqSCwpNCslkepaOO7dTejFRz5thaGiTUBX1p4fZp6sDPDuJrtmNt6Wse9hMpTOF"
	// data := fmt.Sprintf("symbol=BTCUSDT&leverage=20&timestamp=%d", timeStamp)

	// // Create a new HMAC by defining the hash type and the key (as byte array)
	// h := hmac.New(sha256.New, []byte(secret))

	// // Write Data to it
	// h.Write([]byte(data))

	// // Get result and encode as hexadecimal string
	// signature := hex.EncodeToString(h.Sum(nil))

	// req, _ := http.NewRequest("POST", fmt.Sprintf("https://fapi.binance.com/fapi/v1/leverage?symbol=BTCUSDT&leverage=20&timestamp=%d&signature=%s", timeStamp, signature), nil)
	// req.Header.Set("Content-Type", "application/json")
	// req.Header.Add("X-MBX-APIKEY", "klGMQA5VZzL5dhi2DuR4agiYgVZaF8gxmQ0ZEuYkyfURRymazrIYtIBd2TtEheRp")
	// client := &http.Client{}

	// response, err := client.Do(req)

	// if err != nil {
	// 	log.Fatalf("An Error Occured %v", err)
	// } else {
	// 	body, _ := ioutil.ReadAll(response.Body)
	// 	log.Println(string(body))
	// }
	_, file, line, _ := runtime.Caller(0)
	go Log("changeInitialLeverage", fmt.Sprintf("<%v> %v", line, file))
}

func newOrder(symbol string, side string, quantity string, price string) []byte {
	// timeStamp := makeTimestamp()

	// secret := "BfqSCwpNCslkepaOO7dTejFRz5thaGiTUBX1p4fZp6sDPDuJrtmNt6Wse9hMpTOF"
	// // data := fmt.Sprintf("symbol=BTCUSDT&side=BUY&type=LIMIT&timeInForce=GTC&quantity=1000&price=69&timestamp=%d", timeStamp)
	// data := fmt.Sprintf("symbol=%s&side=%s&type=LIMIT&timeInForce=GTC&quantity=%s&price=%s&timestamp=%d", symbol, side, quantity, price, timeStamp)

	// // Create a new HMAC by defining the hash type and the key (as byte array)
	// h := hmac.New(sha256.New, []byte(secret))

	// // Write Data to it
	// h.Write([]byte(data))

	// // Get result and encode as hexadecimal string
	// signature := hex.EncodeToString(h.Sum(nil))

	// req, _ := http.NewRequest("POST", fmt.Sprintf("https://fapi.binance.com/fapi/v1/order?%s&signature=%s", data, signature), nil)
	// req.Header.Set("Content-Type", "application/json")
	// req.Header.Add("X-MBX-APIKEY", "klGMQA5VZzL5dhi2DuR4agiYgVZaF8gxmQ0ZEuYkyfURRymazrIYtIBd2TtEheRp")
	// client := &http.Client{}

	// response, err := client.Do(req)

	// if err != nil {
	// 	log.Fatalf("An Error Occured %v", err)
	// 	return nil
	// } else {
	// 	body, _ := ioutil.ReadAll(response.Body)
	// 	log.Println(string(body))
	// 	return body
	// }
	_, file, line, _ := runtime.Caller(0)
	go Log("newOrder", fmt.Sprintf("<%v> %v", line, file))
	return nil
}

func startUserDataStream() {
	// req, _ := http.NewRequest("POST", "https://fapi.binance.com/fapi/v1/listenKey", nil)
	// req.Header.Set("Content-Type", "application/json")
	// req.Header.Add("X-MBX-APIKEY", "klGMQA5VZzL5dhi2DuR4agiYgVZaF8gxmQ0ZEuYkyfURRymazrIYtIBd2TtEheRp")
	// client := &http.Client{}

	// response, err := client.Do(req)

	// if err != nil {
	// 	log.Fatalf("An Error Occured %v", err)
	// } else {
	// 	body, _ := ioutil.ReadAll(response.Body)
	// 	log.Println(string(body))
	// }
	// fmt.Println("newOrder")
	// return nil
	_, file, line, _ := runtime.Caller(0)
	go Log("startUserDataStream", fmt.Sprintf("<%v> %v", line, file))
}

func accountTradeList() {
	// timeStamp := makeTimestamp()

	// secret := "BfqSCwpNCslkepaOO7dTejFRz5thaGiTUBX1p4fZp6sDPDuJrtmNt6Wse9hMpTOF"
	// data := fmt.Sprintf("symbol=BTCUSDT&limit=500&timestamp=%d", timeStamp)

	// // Create a new HMAC by defining the hash type and the key (as byte array)
	// h := hmac.New(sha256.New, []byte(secret))

	// // Write Data to it
	// h.Write([]byte(data))

	// // Get result and encode as hexadecimal string
	// signature := hex.EncodeToString(h.Sum(nil))

	// req, _ := http.NewRequest("GET", fmt.Sprintf("https://fapi.binance.com/fapi/v1/userTrades?symbol=BTCUSDT&limit=500&timestamp=%d&signature=%s", timeStamp, signature), nil)
	// req.Header.Set("Content-Type", "application/json")
	// req.Header.Add("X-MBX-APIKEY", "klGMQA5VZzL5dhi2DuR4agiYgVZaF8gxmQ0ZEuYkyfURRymazrIYtIBd2TtEheRp")
	// client := &http.Client{}

	// response, err := client.Do(req)

	// if err != nil {
	// 	log.Fatalf("An Error Occured %v", err)
	// } else {
	// 	body, _ := ioutil.ReadAll(response.Body)
	// 	log.Println(string(body))
	// }
	_, file, line, _ := runtime.Caller(0)
	go Log("accountTradeList", fmt.Sprintf("<%v> %v", line, file))
}

func cancelAllOpenOrders(symbol string) []byte {
	// timeStamp := makeTimestamp()

	// secret := "BfqSCwpNCslkepaOO7dTejFRz5thaGiTUBX1p4fZp6sDPDuJrtmNt6Wse9hMpTOF"
	// data := fmt.Sprintf("symbol=%s&timestamp=%d", symbol, timeStamp)

	// // Create a new HMAC by defining the hash type and the key (as byte array)
	// h := hmac.New(sha256.New, []byte(secret))

	// // Write Data to it
	// h.Write([]byte(data))

	// // Get result and encode as hexadecimal string
	// signature := hex.EncodeToString(h.Sum(nil))

	// req, _ := http.NewRequest("DELETE", fmt.Sprintf("https://fapi.binance.com/fapi/v1/allOpenOrders?%s&signature=%s", data, signature), nil)
	// req.Header.Set("Content-Type", "application/json")
	// req.Header.Add("X-MBX-APIKEY", "klGMQA5VZzL5dhi2DuR4agiYgVZaF8gxmQ0ZEuYkyfURRymazrIYtIBd2TtEheRp")
	// client := &http.Client{}

	// response, err := client.Do(req)

	// if err != nil {
	// 	log.Fatalf("An Error Occured %v", err)
	// 	return nil
	// } else {
	// 	body, _ := ioutil.ReadAll(response.Body)
	// 	log.Println(string(body))
	// 	return body
	// }
	_, file, line, _ := runtime.Caller(0)
	go Log("cancelAllOpenOrders", fmt.Sprintf("<%v> %v", line, file))
	return nil
}
