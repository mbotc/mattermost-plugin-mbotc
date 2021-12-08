package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"

	"github.com/pkg/errors"
)

const (
	botUserName    = "mbotc"
	botDisplayName = "MBotC"
	botDescription = "Created by the MBotC plugin."
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// botUserID of the created bot account.
	botUserID string

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	// KV store
	store Store
}

// OnActivate is invoked when the plugin is activated. If an error is returned, the plugin
// will be terminated. The plugin will not receive hooks until after OnActivate returns
// without error. OnConfigurationChange will be called once before OnActivate.
// OnActivate checks if the configurations is valid and ensures the bot account exists
func (p *Plugin) OnActivate() error {
	// create a bot
	botUserID, err := p.Helpers.EnsureBot(&model.Bot{
		Username:    botUserName,
		DisplayName: botDisplayName,
		Description: botDescription,
	})

	if err != nil {
		return errors.Wrap(err, "failed to create bot account")
	}
	// allocate botUserID if success to create bot account
	p.botUserID = botUserID

	// GetBundlePath returns the absolute path where the plugin's bundle was unpacked.
	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		return errors.Wrap(err, "couldn't get bundle path")
	}

	// get bundlePath/assets/profile.png
	profileImage, err := ioutil.ReadFile(filepath.Join(bundlePath, "assets", "profile.png"))
	if err != nil {
		return errors.Wrap(err, "couldn't read profile image")
	}

	// Set bot profile image
	if appErr := p.API.SetProfileImage(botUserID, profileImage); appErr != nil {
		return errors.Wrap(appErr, "couldn't set profile image")
	}

	p.store = NewStore(p)

	// getCommand() of command.go
	command, err := p.getCommand()
	if err != nil {
		return errors.Wrap(err, "failed to get command")
	}

	err = p.API.RegisterCommand(command)
	if err != nil {
		return errors.WithMessage(err, "OnActivate: failed to register command")
	}

	return nil
}

type Notice struct {
	UserId    string   `json:"user_id"`
	Message   string   `json:"message"`
	StartTime string   `json:"start_time"`
	EndTime   string   `json:"end_time"`
	FileIds   []string `json:"file_ids"`
	ChannelId string   `json:"channel_id"`
	PostId    string   `json:"post_id"`
}

type DialogForm struct {
	Type       string `json:"type"`
	CallbackId string `json:"callback_id"`
	State      string `json:"state"`
	UserId     string `json:"user_id"`
	ChannelId  string `json:"channel_id"`
	TeamId     string `json:"team_id"`
	Submission Sub    `json:"submission"`
	Cancelled  bool   `json:"cancelled"`
}

type Sub struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Content   string `json:"content"`
}

func ConvertRequest(p *Plugin, r *http.Request) Notice {
	var notice Notice

	r.ParseMultipartForm(32 << 20) // maxMemory 32MB
	notice.UserId = r.PostFormValue("user_id")
	notice.Message = r.PostFormValue("message")
	notice.StartTime = r.PostFormValue("start_time")
	notice.EndTime = r.PostFormValue("end_time")
	if notice.EndTime == "" {
		notice.EndTime = notice.StartTime
	}
	notice.ChannelId = r.PostFormValue("channel_id")

	fileheaders := r.MultipartForm.File["file"]
	for _, fileheader := range fileheaders {
		file, err := fileheader.Open()
		if err != nil {
			fmt.Println("fileheader.Open() : ", err)
		}
		bytefile, err := ConvertFileToByte(file)
		if err != nil {
			fmt.Println("ConvertRequest Error : ", err)
		}
		notice.FileIds = append(notice.FileIds, UploadFileToMMChannel(p, bytefile, notice.ChannelId, fileheader.Filename))
	}

	return notice
}

func ConvertDialogForm(p *Plugin, r *http.Request) (Notice, error) {
	var notice Notice
	var dialogForm DialogForm

	err := json.NewDecoder(r.Body).Decode(&dialogForm)
	if err != nil {
		fmt.Println("ConvertDialogForm Error : ", err)
		panic(err)
	}

	notice.UserId = dialogForm.UserId
	notice.Message = dialogForm.Submission.Content

	notice.StartTime = dialogForm.Submission.StartTime
	if dialogForm.Submission.EndTime == "" {
		notice.EndTime = dialogForm.Submission.StartTime
	} else {
		notice.EndTime = dialogForm.Submission.EndTime
	}
	notice.ChannelId = dialogForm.ChannelId
	re := regexp.MustCompile(`^\d{4}-(0[1-9]|1[012])-(0[1-9]|[12][0-9]|3[01])\s([01][0-9]|2[0-3]):([012345][0-9])$`)
	if !re.MatchString(notice.StartTime) || !re.MatchString(notice.EndTime) {
		return notice, errors.New("Validation Failed")
	}
	return notice, nil
}

func SendErrorMessage(p *Plugin, notice Notice) {
	if notice.StartTime == notice.EndTime {
		notice.EndTime = ""
	}

	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: notice.ChannelId,
		Message: "Oops! Failed to Create Notice.\n" +
			"Your Input: \n" +
			"\nDate: " + notice.StartTime +
			"\nEnd date: " + notice.EndTime +
			"\nContent: " + notice.Message,
	}
	_ = p.API.SendEphemeralPost(notice.UserId, post)
}

func ConvertFileToByte(file multipart.File) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func UploadFileToMMChannel(p *Plugin, file []byte, channelId string, fileName string) string {
	res, err := p.API.UploadFile(file, channelId, fileName)
	if err != nil {
		fmt.Println("UploadFile Error : ", err)
	}
	return res.Id
}

// ServeHTTP demonstrates a plugin that handles HTTP requests by greeting the world.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	var notice Notice
	// 1. Convert request body to Notice
	switch r.URL.Path {
	case "/fe":
		notice = ConvertRequest(p, r)
	case "/mm":
		var err error
		notice, err = ConvertDialogForm(p, r)
		if err != nil {
			fmt.Print(err)
			SendErrorMessage(p, notice)
			return
		}
	default:
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// 2. Create post (Mattermost)
	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: notice.ChannelId,
		FileIds:   notice.FileIds,
	}
	attachment, err := asSlackAttachment(p, notice)
	if err != nil {
		fmt.Print("asSlackAttachment error : ", err)
	}
	post.AddProp("attachments", attachment)

	resPost, appErr := p.API.CreatePost(post)
	if appErr != nil {
		http.Error(w, appErr.Error(), http.StatusInternalServerError)
		return
	}
	notice.PostId = resPost.Id

	// 3. Send Request to BackEnd if successfully create Post(mattermost)
	siteURL := *p.API.GetConfig().ServiceSettings.SiteURL

	requestUrl := siteURL + ":8080/api/v1/notification"
	noticeJSON, err := json.Marshal(notice)
	if err != nil {
		fmt.Println(err)
	}

	resp, err := http.Post(requestUrl, "application/json", bytes.NewBuffer(noticeJSON))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
}

// See https://developers.mattermost.com/extend/plugins/server/reference/
func asSlackAttachment(p *Plugin, notice Notice) ([]*model.SlackAttachment, error) {
	var text = notice.Message
	var fields []*model.SlackAttachmentField

	teamName, channelName := SearchTeamNameAndChannelName(p, notice.ChannelId)

	var postBy = teamName + " / " + channelName

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

	user, _ := p.API.GetUser(notice.UserId)
	fields = append(fields, &model.SlackAttachmentField{
		Title: ":lower_left_fountain_pen: Author",
		Value: user.Username,
		Short: false,
	})

	// 작성자 이름, 기간시작(yyyy-mm-dd hh:mm), 기간끝, 컨텐츠, 팀, 채널
	return []*model.SlackAttachment{
		{
			AuthorName: postBy,
			Color:      "#1352ab",
			Text:       text,
			Fields:     fields,
		},
	}, nil
}

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
