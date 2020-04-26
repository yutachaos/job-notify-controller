package notification

import (
	"bytes"
	"html/template"
	"os"

	slackapi "github.com/slack-go/slack"
	"k8s.io/klog"
)

const (
	START                = "start"
	SUCCESS              = "success"
	FAILED               = "failed"
	SlackMessageTemplate = `
*JobName*: {{.JobName}}
{{if .Namespace}} *Namespace*: {{.Namespace}} {{end}}
{{if .Log }} *Loglink*: {{.Log}} {{end}}
`
)

var slackColors = map[string]string{
	"Normal":  "good",
	"Warning": "warning",
	"Danger":  "danger",
}

type slack struct {
	token    string
	channel  string
	username string
}

type MessageTemplateParam struct {
	JobName   string
	Namespace string
	Log       string
}

type Slack interface {
	NotifyStart(messageParam MessageTemplateParam) (err error)
	NotifySuccess(messageParam MessageTemplateParam) (err error)
	NotifyFailed(messageParam MessageTemplateParam) (err error)
	notify(attachment slackapi.Attachment) (err error)
}

func NewSlack() Slack {
	token := os.Getenv("SLACK_TOKEN")
	if token == "" {
		panic("please set slack token")
	}

	channel := os.Getenv("SLACK_CHANNEL")

	username := os.Getenv("SLACK_USERNAME")

	return slack{
		token:    token,
		channel:  channel,
		username: username,
	}

}

func (s slack) NotifyStart(messageParam MessageTemplateParam) (err error) {

	succeedChannel := os.Getenv("SLACK_SUCCEED_CHANNEL")
	if succeedChannel != "" {
		s.channel = succeedChannel
	}

	slackMessage, err := getSlackMessage(messageParam)
	if err != nil {
		klog.Errorf("Template execute failed %s\n", err)
		return err
	}

	attachment := slackapi.Attachment{
		Color: slackColors["Normal"],
		Title: "Job Start",
		Text:  slackMessage,
	}

	err = s.notify(attachment)
	if err != nil {
		return err
	}
	return nil
}

func getSlackMessage(messageParam MessageTemplateParam) (slackMessage string, err error) {
	var b bytes.Buffer
	tpl, err := template.New("slack").Parse(SlackMessageTemplate)
	if err != nil {
		return "", err
	}
	err = tpl.Execute(&b, messageParam)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

func (s slack) NotifySuccess(messageParam MessageTemplateParam) (err error) {
	succeedChannel := os.Getenv("SLACK_SUCCEED_CHANNEL")
	if succeedChannel != "" {
		s.channel = succeedChannel
	}
	if messageParam.Log != "" {
		file, err := s.uploadLog(messageParam)
		if err != nil {
			klog.Errorf("Template execute failed %s\n", err)
			return err
		}
		messageParam.Log = file.Permalink
	}

	slackMessage, err := getSlackMessage(messageParam)
	if err != nil {
		klog.Errorf("Template execute failed %s\n", err)
		return err
	}
	attachment := slackapi.Attachment{
		Color: slackColors["Normal"],
		Title: "Job Success",
		Text:  slackMessage,
	}

	err = s.notify(attachment)
	if err != nil {
		return err
	}
	return nil
}

func (s slack) NotifyFailed(messageParam MessageTemplateParam) (err error) {
	failedChannel := os.Getenv("SLACK_FAILED_CHANNEL")
	if failedChannel != "" {
		s.channel = failedChannel
	}
	if messageParam.Log != "" {
		file, err := s.uploadLog(messageParam)
		if err != nil {
			klog.Errorf("Template execute failed %s\n", err)
			return err
		}
		messageParam.Log = file.Permalink
	}

	slackMessage, err := getSlackMessage(messageParam)
	if err != nil {
		klog.Errorf("Template execute failed %s\n", err)
		return err
	}

	attachment := slackapi.Attachment{
		Color: slackColors["Danger"],
		Title: "Job Failed",
		Text:  slackMessage,
	}

	err = s.notify(attachment)
	if err != nil {
		return err
	}
	return nil
}

func (s slack) notify(attachment slackapi.Attachment) (err error) {
	api := slackapi.New(s.token)

	channelID, timestamp, err := api.PostMessage(
		s.channel,
		slackapi.MsgOptionText("", true),
		slackapi.MsgOptionAttachments(attachment),
		slackapi.MsgOptionUsername(s.username),
	)

	if err != nil {
		klog.Errorf("Send messageParam failed %s\n", err)
		return
	}

	klog.Infof("Message successfully sent to channel %s at %s", channelID, timestamp)
	return err
}

func (s slack) uploadLog(param MessageTemplateParam) (file *slackapi.File, err error) {
	api := slackapi.New(s.token)

	file, err = api.UploadFile(
		slackapi.FileUploadParameters{
			Title:    param.Namespace + "_" + param.JobName,
			Content:  param.Log,
			Filetype: "txt",
			Channels: []string{s.channel},
		})
	if err != nil {
		klog.Errorf("File uploadLog failed %s\n", err)
		return
	}

	klog.Infof("File uploadLog successfully %s", file.Name)
	return
}
