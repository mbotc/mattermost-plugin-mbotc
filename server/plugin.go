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
	UserName  string   `json:"user_name"`
	Message   string   `json:"message"`
	Time      string   `json:"time"`
	StartTime string   `json:"start_time"`
	EndTime   string   `json:"end_time"`
	FileIds   []string `json:"file_ids"`
	ChannelId string   `json:"channel_id"`
}

func ConvertRequest(p *Plugin, r *http.Request) Notice {
	var notice Notice

	r.ParseMultipartForm(32 << 20) // maxMemory 32MB
	notice.UserId = r.PostFormValue("user_id")
	notice.UserName = r.PostFormValue("user_name")
	notice.Message = r.PostFormValue("message")
	notice.Time = r.PostFormValue("time")
	notice.StartTime = r.PostFormValue("start_time")
	notice.EndTime = r.PostFormValue("end_time")
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
	// 1. FE에서 받아와서
	notice := ConvertRequest(p, r)

	// 2. MM에 create post
	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: notice.ChannelId,
		FileIds:   notice.FileIds,
	}

	attachment, err := asSlackAttachment(p, notice)
	if err != nil {
		fmt.Print(err)
	}
	post.AddProp("attachments", attachment)

	res, appErr := p.API.CreatePost(post)
	if appErr != nil {
		http.Error(w, appErr.Error(), http.StatusInternalServerError)
		return
	}

	// 3. post 성공하면 그 내용을 BE로 요청
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.Encode(res)
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

	fields = append(fields, &model.SlackAttachmentField{
		Title: ":lower_left_fountain_pen: Author",
		Value: notice.UserName,
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
	channel, _ := p.API.GetChannel(channelId)
	team, _ := p.API.GetTeam(channel.TeamId)

	return team.DisplayName, channel.DisplayName
}
