package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/v5/model"
)

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

func convertDialogForm(p *Plugin, r *http.Request) (Notice, error) {
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

func convertRequest(p *Plugin, r *http.Request) Notice {
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

func (p *Plugin) httpCreateNoticeWithCommand(w http.ResponseWriter, r *http.Request) {
	var err error
	notice, err := convertDialogForm(p, r)
	if err != nil {
		fmt.Print(err)
		post := getConvertErrorPost(p, notice)
		p.API.SendEphemeralPost(notice.UserId, post)
		return
	}
	p.httpCreatePost(w, notice)
}

func (p *Plugin) httpCreateNoticeWithEditor(w http.ResponseWriter, r *http.Request) {
	notice := convertRequest(p, r)
	p.httpCreatePost(w, notice)
}

func (p *Plugin) httpCreatePost(w http.ResponseWriter, notice Notice) {
	post := &model.Post{
		UserId:    p.botUserID,
		ChannelId: notice.ChannelId,
		FileIds:   notice.FileIds,
	}
	attachment, err := asSlackAttachment(p, notice)
	if err != nil {
		fmt.Print("asSlackAttachment error ", err)
	}
	post.AddProp("attachments", attachment)

	resPost, appErr := p.API.CreatePost(post)
	if appErr != nil {
		http.Error(w, appErr.Error(), http.StatusInternalServerError)
		return
	}
	notice.PostId = resPost.Id

	requestUrl := serverUrl + "/api/v1/notification"
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

func getConvertErrorPost(p *Plugin, notice Notice) *model.Post {
	if notice.StartTime == notice.EndTime {
		notice.EndTime = ""
	}

	return &model.Post{
		UserId:    p.botUserID,
		ChannelId: notice.ChannelId,
		Message: "Oops! Failed to Create Notice.\n" +
			"Your Input: \n" +
			"\nDate: " + notice.StartTime +
			"\nEnd date: " + notice.EndTime +
			"\nContent: " + notice.Message,
	}
}

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
