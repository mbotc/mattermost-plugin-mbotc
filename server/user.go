package main

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

func checkAuthentication(p *Plugin, userId string, channelId string) error {
	resp, err := checkUserExists(p, userId)

	if err != nil {
		p.postEphemeralResponse(userId, channelId, "Oops! Something wrong")
		return err
	}

	if resp.StatusCode == 404 {
		var message = "Please login first to use MBotC service.\n" +
		"[Login here](" + clientUrl + ")"
		p.postEphemeralResponse(userId, channelId, message)
		return errors.New("USER NOT FOUND")
	}

	return nil
}

// Check if user exists in Database
func checkUserExists(p *Plugin, userId string) (*http.Response, error) {
	requestUrl := serviceAPIUrl + "/api/v1/user"
	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		fmt.Println("NewRequest Error: ", err)
		panic(err)
	}
	req.Header.Add("userId", userId)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("client.Do Error: ", err)
	}
	defer resp.Body.Close()

	return resp, err
}

func getAuthor(p *Plugin, userId string) string {
	user, _ := p.API.GetUser(userId)
	var author string
	if user.Nickname != "" {
		author = user.Nickname
	} else {
		author = user.Username
	}
	return author
}
