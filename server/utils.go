package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/mattermost/mattermost-server/v5/model"
)

// Post Message to Channel with Bot
func (p *Plugin) postEphemeralResponse(userId string, channelId string, message string) {
	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: channelId,
		Message:   message,
	}
	_ = p.API.SendEphemeralPost(userId, post)
}

// Convert multipart file to []byte
func ConvertFileToByte(file multipart.File) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Upload file to mattermost channel and return file id
func UploadFileToMMChannel(p *Plugin, file []byte, channelId string, fileName string) string {
	resp, err := p.API.UploadFile(file, channelId, fileName)
	if err != nil {
		fmt.Println("UploadFile Error: ", err)
	}
	return resp.Id
}

// Get teamName and channelName by channelId
func SearchTeamNameAndChannelName(p *Plugin, channelId string) (teamName string, channelName string) {
	channel, err := p.API.GetChannel(channelId)
	if err != nil {
		fmt.Print("GetChannel Error", err)
	}
	team, err := p.API.GetTeam(channel.TeamId)
	if err != nil {
		fmt.Print("GetTeam Error", err)
	}

	return team.DisplayName, channel.DisplayName
}
