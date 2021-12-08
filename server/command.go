package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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
		AutoCompleteDesc:     "Available commands: help, create",
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
		"help":   executeHelp,
		"create": executeCreate,
		"today":  executeToday,
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
	var helpText = "###### Mattermost MBotC Plugin - Slash Command Help\n" +
		"* `/mbotc help` - help text\n" +
		"* `/mbotc create` - Create your Notice\n" +
		" File Upload is not supported\n" +
		" If you want to upload file, please visit [here](" + clientUrl + ")\n"
	p.postCommandResponse(header, helpText)
	return &model.CommandResponse{}
}

func executeCreate(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	p.openCreateDialog(header)
	return &model.CommandResponse{}
}

func executeToday(p *Plugin, c *plugin.Context, header *model.CommandArgs, args ...string) *model.CommandResponse {
	getNoticeList(p, header)
	return &model.CommandResponse{}
}

func getNoticeList(p *Plugin, commandArgs *model.CommandArgs) {
	requestUrl := serviceAPIUrl + "/api/v1/notification/today"
	// create new request
	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		fmt.Println("NewRequest Error: ", err)
		panic(err)
	}

	// set the header
	req.Header.Add("userId", commandArgs.UserId)

	client := &http.Client{}
	resp, err := client.Do(req) // send request
	if err != nil {
		fmt.Println("client.Do Error: ", err)
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

	var text string

	text = "# Today's Notice\n" +
		"| Preview :loudspeaker: | Deadline :calendar: |\n" +
		"| --- | --- |\n"

	if len(dailyNotices) == 0 {
		text += "| Nothing ... | - |\n"
	} else {
		for _, dn := range dailyNotices {
			var message = strings.Replace(dn.Message, "\n", " ", -1)
			if len(message) >= 100 {
				message = message[:100] + " ..."
			}
			text += "| " + message + " | " + dn.EndTime + " | \n"
		}
	}

	loc, _ := time.LoadLocation("Asia/Seoul")
	currentTime := time.Now().In(loc)
	text += "[See More](" + clientUrl + "/main/detail/" + currentTime.Format("20060102") + ")"
	var attachment = []*model.SlackAttachment{
		{
			Color: "#1352ab",
			Text:  text,
		},
	}
	post.AddProp("attachments", attachment)

	_ = p.API.SendEphemeralPost(commandArgs.UserId, post)
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

	create := model.NewAutocompleteData("create", "", "Register your Notice")
	mbotcAutocomplete.AddCommand(create)

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

func (p *Plugin) openCreateDialog(args *model.CommandArgs) {
	siteURL := *p.API.GetConfig().ServiceSettings.SiteURL
	dialogRequest := model.OpenDialogRequest{
		TriggerId: args.TriggerId,
		URL:       fmt.Sprintf("%s/plugins/%s/api/v1/create-notice-with-command", siteURL, "com.mattermost.plugin-mbotc"),
		Dialog:    getDialog(),
	}

	p.API.OpenInteractiveDialog(dialogRequest)
}

func getDialog() model.Dialog {
	loc, _ := time.LoadLocation("Asia/Seoul")
	currentTime := time.Now().In(loc)

	return model.Dialog{
		CallbackId: "somecallbackid",
		Title:      "Create Notice",
		Elements: []model.DialogElement{{
			DisplayName: "Date",
			Name:        "start_time",
			Type:        "text",
			Placeholder: "YYYY-MM-DD hh:mm",
			Default:     currentTime.Format("2006-01-02 15:04"),
			HelpText:    "e.g. 2021-11-05 09:00",
		}, {
			DisplayName: "End date",
			Name:        "end_time",
			Type:        "text",
			Optional:    true,
			Placeholder: "YYYY-MM-DD hh:mm",
			HelpText:    "e.g. 2021-11-05 18:00",
		}, {
			DisplayName: "Content",
			Name:        "content",
			Type:        "textarea",
			Placeholder: "Write what you want to notice",
			HelpText:    "Write in Markdown syntax.",
		}},
		SubmitLabel:    "Submit",
		NotifyOnCancel: false,
	}
}
