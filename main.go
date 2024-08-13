package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type RegisterData struct {
	AccessRequestCount int                    `json:"accessRequestCount"`
	AccessToken        string                 `json:"accessToken"`
	Username           string                 `json:"username"`
	Email              string                 `json:"email"`
	Admin              bool                   `json:"admin"`
	Avatar             map[string]interface{} `json:"avatar"`
	FollowerCount      int                    `json:"followerCount"`
	FollowingCount     int                    `json:"followingCount"`
	Info               map[string]interface{} `json:"info"`
}

type RegisterResponse struct {
	Data    RegisterData `json:"data"`
	Status  string       `json:"status"`
	Message string       `json:"message"`
	Fields  interface{}  `json:"fields"`
}

type GetBalanceResponse struct {
	Data    float64     `json:"data"`
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Fields  interface{} `json:"fields"`
}

func GetBalanceRequest(accessToken string, tokenName string) (*GetBalanceResponse, time.Duration) {
	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:8080/api/user/balance?token=%s", tokenName), nil)
	if err != nil {
		log.Printf("Error creating request: %v\n", err)
		return nil, 0
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}

	startTime := time.Now()

	httpResponse, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending request: %v\n", err)
		return nil, 0
	}
	defer httpResponse.Body.Close()

	duration := time.Since(startTime)

	var response GetBalanceResponse
	if err = json.NewDecoder(httpResponse.Body).Decode(&response); err != nil {
		return nil, 0
	}
	return &response, duration
}

func RegisterNewUser(data map[string]interface{}) (*RegisterResponse, time.Duration) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshalling data: %v\n", err)
		return nil, 0
	}
	req, err := http.NewRequest("POST", "http://localhost:8080/api/auth/register", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error creating request: %v\n", err)
		return nil, 0
	}
	req.Header.Set("Content-Type", "application/json")
	startTime := time.Now()
	client := &http.Client{}
	httpResponse, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending request: %v\n", err)
		return nil, 0
	}
	defer httpResponse.Body.Close()

	duration := time.Since(startTime)
	var response RegisterResponse
	if err = json.NewDecoder(httpResponse.Body).Decode(&response); err != nil {
		return nil, 0
	}
	return &response, duration
}

func RegisterRandomUsers(count int) ([]*RegisterResponse, time.Duration, time.Duration) {
	users := make([]*RegisterResponse, count, count)
	var waiter sync.WaitGroup
	startTime := time.Now()
	totalSumDuration := time.Duration(0)
	for i := 0; i < count; i++ {
		waiter.Add(1)
		go func() {
			key := fmt.Sprintf("unix%d%d", time.Now().Unix(), i)
			registerationData := map[string]interface{}{
				"email":            key + "@gmail.com",
				"username":         key,
				"password":         "Un1x_Generated",
				"avatarId":         1,
				"verificationCode": "12345",
				"referralCode":     "",
			}
			resp, duration := RegisterNewUser(registerationData)
			users[i] = resp
			totalSumDuration += duration
			waiter.Done()
		}()
	}

	waiter.Wait()
	fullDuration := time.Since(startTime)
	return users, fullDuration, totalSumDuration
}

func LogPerformanceTest(users []*RegisterResponse, balances []*GetBalanceResponse, actualDuration time.Duration, sumDuration time.Duration) {
	filename := fmt.Sprintf("log_%d.csv", time.Now().Unix())

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	length := min(len(users), len(balances)) // just to make sure

	if _, err := file.WriteString("Username,Balance,ActualDuration,SumDuration\n"); err != nil {
		log.Printf("Error writing header to csv file: %v\n", err)
		return
	}
	// Write users to the file
	for i := 0; i < length; i++ {
		logText := fmt.Sprintf("%s, %f, %v, %v\n", users[i].Data.Username, balances[i].Data, actualDuration, sumDuration)
		if _, err := file.WriteString(logText); err != nil {
			log.Printf("Error writing to file: %v\n", err)
			return
		}
	}

	log.Println("Data written to file successfully.")
}

func main() {
	const (
		TOTAL_COUNT = 500
		GAS_NAME    = "gas"
	)
	users, actualDuration, sumDuration := RegisterRandomUsers(TOTAL_COUNT)
	log.Println(TOTAL_COUNT, "Users registered in ", actualDuration, " And sumDuration = ", sumDuration)
	for {
		balances := make([]*GetBalanceResponse, TOTAL_COUNT, TOTAL_COUNT)
		var waiter sync.WaitGroup
		sumDuration = 0
		startTime := time.Now()
		for i, user := range users {
			waiter.Add(1)
			go func() {
				resp, duration := GetBalanceRequest(user.Data.AccessToken, GAS_NAME)
				sumDuration += duration
				balances[i] = resp
				waiter.Done()
			}()
		}
		waiter.Wait()
		actualDuration = time.Since(startTime)
		LogPerformanceTest(users, balances, actualDuration, sumDuration)
		time.Sleep(20)
	}
}
