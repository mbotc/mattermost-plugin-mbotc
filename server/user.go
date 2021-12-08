package main

import (
	"fmt"
	"net/http"
)

func checkUserExists(p *Plugin, userId string) (*http.Response, error) {
	requestUrl := serviceAPIUrl + "/api/v1/user"
	// create new request
	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		fmt.Println("NewRequest Error: ", err)
		panic(err)
	}

	// set the header
	req.Header.Add("userId", userId)

	client := &http.Client{}
	resp, err := client.Do(req) // send request
	if err != nil {
		fmt.Println("client.Do Error: ", err)
		// panic(err)
	}
	defer resp.Body.Close()

	return resp, err
}
