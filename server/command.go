package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost-plugin-api/experimental/command"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
	"github.com/pkg/errors"
)

func (p *Plugin) getCommand() (*model.Command, error) {
	iconData, err := command.GetIconData(p.API, "assets/mbotc-icon.svg")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get icon data")
	}

	return &model.Command{
		Trigger:              "mbotc",
		DisplayName:          "mbotc",
		Description:          "Integration with MBotC.",
		AutoComplete:         true,
		AutoCompleteDesc:     "Available commands: help, term, date",
		AutoCompleteHint:     "[command]",
		AutocompleteData:     getAutocompleteData(),
		AutocompleteIconData: iconData,
	}, nil
}

type DailyNotice struct {
	ChannelName string `json:"channel_name"`
	EndTime     string `json:"end_time"`
	Message     string `json:"message"`
	StartTime   string `json:"start_time"`
	TeamName    string `json:"team_name"`
	UserName    string `json:"user_name"`
}

const helpText = "###### Mattermost MBotC Plugin - Slash Command Help\n" +
	"* `/mbotc help` - help text\n" +
	"* `/mbotc term` - Register your Term Notice.\n" +
	"	```\n" +
	"	[Template]\n" +
	"	<write here what you want to notice with markdown format>\n" +
	"	`date YYYY-MM-DD hh:mm - YYYY-MM-DD hh:mm\n" +
	"	```\n" +
	"* `/mbotc date` - Register your Date Notice.\n" +
	"	```\n" +
	"	[Template]\n" +
	"	<write here what you want to notice with markdown format>\n" +
	"	`date YYYY-MM-DD hh:mm\n" +
	"	```\n"

type CommandHandlerFunc func(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse

type CommandHandler struct {
	handlers       map[string]CommandHandlerFunc
	defaultHandler CommandHandlerFunc
}

//===================================================
// command Handler
// command : func
//===================================================
var mbotcCommandHandler = CommandHandler{
	handlers: map[string]CommandHandlerFunc{
		"help":  executeHelp,
		"term":  executeTerm,
		"date":  executeDate,
		"today": executeToday,
	},
	defaultHandler: executeHelp,
}

func (ch CommandHandler) Handle(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	for n := len(args); n > 0; n-- {
		h := ch.handlers[strings.Join(args[:n], "/")]
		if h != nil {
			return h(p, c, header, args[n:]...)
		}
	}
	return ch.defaultHandler(p, c, header, args...)
}

func executeHelp(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	return p.help(header)
}

func (p *Plugin) help(header *model.CommandArgs) *model.CommandResponse {
	p.postCommandResponse(header, helpText)
	return &model.CommandResponse{}
}

func executeTerm(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	p.postCommandResponse(header, "###### Successfully registered your term notice:\n"+header.Command)
	p.registerNotice(header)
	return &model.CommandResponse{}
}

func executeDate(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	p.postCommandResponse(header, "###### Successfully registered your term notice:\n"+header.Command)
	p.registerNotice(header)
	return &model.CommandResponse{}
}

func executeToday(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	getNoticeList(p, header)
	return &model.CommandResponse{}
}

func getNoticeList(p *Plugin, commandArgs *model.CommandArgs) {
	// create new request
	req, err := http.NewRequest("GET", "http://localhost:8080/api/v1/notification/today", nil)
	if err != nil {
		// panic()함수는 현재 함수를 즉시 멈추고 현재 함수에 defer 함수들을 모두 실행한 후 즉시 리턴한다
		fmt.Println(err)
		panic(err)
	}

	// set the header
	req.Header.Add("auth", commandArgs.UserId)

	client := &http.Client{}
	resp, err := client.Do(req) // send request
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	defer resp.Body.Close()

	bytes, _ := ioutil.ReadAll(resp.Body)

	var dailyNotices []DailyNotice
	json.Unmarshal(bytes, &dailyNotices)

	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: commandArgs.ChannelId,
	}

	for _, dn := range dailyNotices {
		attachment, err := createAttachment(dn)
		if err != nil {
			fmt.Print(err)
		}
		post.AddProp("attachments", attachment)
	}

	_ = p.API.SendEphemeralPost(commandArgs.UserId, post)
}

// Send Post Request to our Bot server
func (p *Plugin) registerNotice(commandArgs *model.CommandArgs) {
	pbytes, _ := json.Marshal(commandArgs)
	buff := bytes.NewBuffer(pbytes)
	// http.Post(url, request body MIME type, data)
	resp, err := http.Post("http://httpbin.org/post", "application/json", buff)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()

	// Check Response
	respBody, err := ioutil.ReadAll(resp.Body)
	if err == nil {
		str := string(respBody)
		fmt.Println(str)
	}
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, commandArgs *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	args := strings.Fields(commandArgs.Command)
	if len(args) == 0 || args[0] != "/mbotc" {
		return p.help(commandArgs), nil
	}

	return mbotcCommandHandler.Handle(p, c, commandArgs, args[1:]...), nil
}

func getAutocompleteData() *model.AutocompleteData {
	mbotcAutocomplete := model.NewAutocompleteData("mbotc", "[command]", "Available commands: help, term, date")

	help := model.NewAutocompleteData("help", "", "Guide for mbotc")
	mbotcAutocomplete.AddCommand(help)

	term := model.NewAutocompleteData("term", "[text]", "Register your Term Notice (Please refer to /help)")
	mbotcAutocomplete.AddCommand(term)

	date := model.NewAutocompleteData("date", "[text]", "Register your Date Notice (Please refer to /help)")
	mbotcAutocomplete.AddCommand(date)

	today := model.NewAutocompleteData("today", "", "Get all today's notices")
	mbotcAutocomplete.AddCommand(today)

	return mbotcAutocomplete
}

// Post Message to Channel with Bot
func (p *Plugin) postCommandResponse(args *model.CommandArgs, text string) {
	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: args.ChannelId,
		Message:   text,
	}
	_ = p.API.SendEphemeralPost(args.UserId, post)
}

func createAttachment(notice DailyNotice) ([]*model.SlackAttachment, error) {
	var text = notice.Message
	var fields []*model.SlackAttachmentField
	var postedBy = notice.TeamName + " / " + notice.ChannelName

	if notice.StartTime == notice.EndTime {
		fields = append(fields, &model.SlackAttachmentField{
			Title: ":calendar: Deadline",
			Value: notice.StartTime,
			Short: false,
		})
	} else {
		fields = append(fields, &model.SlackAttachmentField{
			Title: ":calendar: Start Time",
			Value: notice.StartTime,
			Short: true,
		})
		fields = append(fields, &model.SlackAttachmentField{
			Title: ":calendar: End Time",
			Value: notice.EndTime,
			Short: true,
		})
	}

	fields = append(fields, &model.SlackAttachmentField{
		Title: ":lower_left_fountain_pen: Author",
		Value: notice.UserName,
		Short: false,
	})

	return []*model.SlackAttachment{
		{
			AuthorName: postedBy,
			Color:      "#1352ab",
			Text:       text,
			Fields:     fields,
		},
	}, nil
}
