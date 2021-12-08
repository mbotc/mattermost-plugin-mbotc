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
		"help": executeHelp,
		"term": executeTerm,
		"date": executeDate,
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
