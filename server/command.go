package main

import (
	"fmt"
	"strings"
	"time"

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
		AutoCompleteDesc:     "Available commands: help, create, today",
		AutoCompleteHint:     "[command]",
		AutocompleteData:     getAutocompleteData(),
		AutocompleteIconData: iconData,
	}, nil
}

type DailyNotification struct {
	TeamName    string `json:"team_name"`
	ChannelName string `json:"channel_name"`
	UserName    string `json:"user_name"`
	Message     string `json:"message"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
}

type CommandHandlerFunc func(p *Plugin, c *plugin.Context, commandArgs *model.CommandArgs, args ...string) *model.CommandResponse

type CommandHandler struct {
	handlers       map[string]CommandHandlerFunc
	defaultHandler CommandHandlerFunc
}

var mbotcCommandHandler = CommandHandler{
	handlers: map[string]CommandHandlerFunc{
		"help":   executeHelp,
		"create": executeCreate,
		"today":  executeToday,
	},
	defaultHandler: executeHelp,
}

func (ch CommandHandler) Handle(p *Plugin, c *plugin.Context, commandArgs *model.CommandArgs, args ...string) *model.CommandResponse {
	for n := len(args); n > 0; n-- {
		h := ch.handlers[strings.Join(args[:n], "/")]
		if h != nil {
			return h(p, c, commandArgs, args[n:]...)
		}
	}
	return ch.defaultHandler(p, c, commandArgs, args...)
}

func executeHelp(p *Plugin, c *plugin.Context, commandArgs *model.CommandArgs, args ...string) *model.CommandResponse {
	return p.help(commandArgs)
}

func (p *Plugin) help(commandArgs *model.CommandArgs) *model.CommandResponse {
	var helpText = "###### Mattermost MBotC Plugin - Slash Command Help\n" +
		"* `/mbotc create` - Create a new Notification with dialog " +
		" (or [here](+ " + clientUrl + ") : you can upload files)\n" +
		"* `/mbotc today` - Will list the notifications you subscribed\n"
	p.postEphemeralResponse(commandArgs.UserId, commandArgs.ChannelId, helpText)
	return &model.CommandResponse{}
}

func executeCreate(p *Plugin, c *plugin.Context, commandArgs *model.CommandArgs, args ...string) *model.CommandResponse {
	err := checkAuthentication(p, commandArgs.UserId, commandArgs.ChannelId)
	if err == nil {
		p.openCreateDialog(commandArgs)
	}
	return &model.CommandResponse{}
}

func executeToday(p *Plugin, c *plugin.Context, commandArgs *model.CommandArgs, args ...string) *model.CommandResponse {
	err := checkAuthentication(p, commandArgs.UserId, commandArgs.ChannelId)
	if err == nil {
		getNotificationList(p, commandArgs)
	}
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
	mbotcAutocomplete := model.NewAutocompleteData("mbotc", "[command]", "Available commands: help, create, today")

	help := model.NewAutocompleteData("help", "", "Guide for mbotc slash command")
	mbotcAutocomplete.AddCommand(help)

	create := model.NewAutocompleteData("create", "", "Register your Notification")
	mbotcAutocomplete.AddCommand(create)

	today := model.NewAutocompleteData("today", "", "Get today's notification list")
	mbotcAutocomplete.AddCommand(today)

	return mbotcAutocomplete
}

// Open form dialog for /mbotc create
func (p *Plugin) openCreateDialog(args *model.CommandArgs) {
	siteURL := *p.API.GetConfig().ServiceSettings.SiteURL
	requestUrl := fmt.Sprintf("%s/plugins/%s/api/v1/create-notification-with-command", siteURL, pluginId)
	dialogRequest := model.OpenDialogRequest{
		TriggerId: args.TriggerId,
		URL:       requestUrl,
		Dialog:    getDialog(),
	}

	p.API.OpenInteractiveDialog(dialogRequest)
}

func getDialog() model.Dialog {
	loc, _ := time.LoadLocation(timezone)
	currentTime := time.Now().In(loc)

	return model.Dialog{
		CallbackId: "somecallbackid",
		Title:      "Create Notification",
		Elements: []model.DialogElement{{
			DisplayName: "Date",
			Name:        "start_time",
			Type:        "text",
			Placeholder: "YYYY-MM-DD hh:mm",
			Default:     currentTime.Format("2006-01-02 15:04"),
			HelpText:    "e.g. 2006-01-02 15:04",
		}, {
			DisplayName: "End date",
			Name:        "end_time",
			Type:        "text",
			Optional:    true,
			Placeholder: "YYYY-MM-DD hh:mm",
			HelpText:    "e.g. 2006-01-02 15:04",
		}, {
			DisplayName: "Message",
			Name:        "message",
			Type:        "textarea",
			Placeholder: "Write notification message",
			HelpText:    "Write in Markdown syntax.",
		}},
		SubmitLabel:    "Submit",
		NotifyOnCancel: false,
	}
}
