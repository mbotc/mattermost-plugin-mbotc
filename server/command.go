package main

import (
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
		AutoCompleteDesc:     "Available commands: help",
		AutoCompleteHint:     "[command]",
		AutocompleteData:     getAutocompleteData(),
		AutocompleteIconData: iconData,
	}, nil
}

const helpText = "###### Mattermost MBotC Plugin - Slash Command Help\n" +
	"* `/mbotc help` - help text"

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
		"help":       executeHelp,
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

func (p *Plugin) ExecuteCommand(c *plugin.Context, commandArgs *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	args := strings.Fields(commandArgs.Command)
	if len(args) == 0 || args[0] != "/mbotc" {
		return p.help(commandArgs), nil
	}

	return mbotcCommandHandler.Handle(p, c, commandArgs, args[1:]...), nil
}

func getAutocompleteData() *model.AutocompleteData {
	webexAutocomplete := model.NewAutocompleteData("mbotc", "[command]", "Available commands: help")

	help := model.NewAutocompleteData("help", "", "Guide for mbotc")
	webexAutocomplete.AddCommand(help)

	return webexAutocomplete
}

// Post Message to Channel
func (p *Plugin) postCommandResponse(args *model.CommandArgs, text string) {
	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: args.ChannelId,
		Message:   text,
	}
	_ = p.API.SendEphemeralPost(args.UserId, post)
}
