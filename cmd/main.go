package main

import (
	"fmt"
	jira "github.com/andygrunwald/go-jira"
	"github.com/davecgh/go-spew/spew"
	_ "github.com/joho/godotenv/autoload"
	"github.com/slack-go/slack"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

var jiraClient *jira.Client

type Worklog struct {
	ValueList []struct {
		WorklogId   int64         `json:"worklogId"`
		UpdatedTime int64         `json:"updatedTime"`
		Properties  []interface{} `json:"properties"`
	} `json:"values"`
	Since    int64  `json:"since"`
	Until    int64  `json:"until"`
	Self     string `json:"self"`
	NextPage string `json:"nextPage"`
	LastPage bool   `json:"lastPage"`
}

type WorklogIds struct {
	Ids []int64 `json:"ids"`
}

type WorklogInfo struct {
	Author struct {
		DisplayName  string `json:"displayName"`
		AccountId    string `json:"accountId"`
		Active       bool   `json:"active"`
		EmailAddress string `json:"emailAddress"`
	} `json:"author"`
	UpdateAuthor struct {
		DisplayName  string `json:"displayName"`
		AccountId    string `json:"accountId"`
		Active       bool   `json:"active"`
		EmailAddress string `json:"emailAddress"`
	} `json:"updateAuthor"`
	TimeSpent        string `json:"timeSpent"`
	TimeSpentSeconds int64  `json:"timeSpentSeconds"`
	Started          string `json:"started"`
}

type Haystack []string

func (haystack Haystack) Has(needle string) bool {
	for _, iterable := range haystack {
		if iterable == needle {
			return true
		}
	}

	return false
}

func (Haystack) Make(string string) Haystack {
	splitString := strings.Split(string, ",")

	haystack := make(Haystack, len(splitString))
	copy(haystack, splitString)

	return haystack
}

var Debug string

func main() {
	Debug = "1"

	tp := jira.BasicAuthTransport{
		Username: os.Getenv("JIRA_API_USERNAME"),
		Password: os.Getenv("JIRA_API_TOKEN"),
	}

	jiraClient, _ = jira.NewClient(tp.Client(), os.Getenv("JIRA_API_BASE_URL"))

	worklogIds := retrieveWorklogIds()
	worklogInfoList := retrieveWorklogInfoList(worklogIds)

	emailsForObservation := Haystack.Make(Haystack{}, os.Getenv("JIRA_EMAILS_FOR_OBSERVATION"))

	sumWorkHoursEachUser := calcSumWorkHoursEachUser(worklogInfoList, emailsForObservation)
	tresholdHours, _ := strconv.Atoi(os.Getenv("JIRA_THRESHOLD_HOURS"))

	fmt.Printf("from %s\n", getSinceDate())

	slackApi := slack.New(os.Getenv("SLACK_TOKEN"))

	attachment := slack.Attachment{
		Pretext: "Time log notification",
		Text:    "Do not forget log time for today",
		Color:   "#FFC700",
		Fields:  []slack.AttachmentField{},
	}

	for email, timeSpentSeconds := range sumWorkHoursEachUser {
		if !emailsForObservation.Has(email) {
			continue
		}

		hoursLogged := float64(timeSpentSeconds) / 60 / 60

		fmt.Printf("%s - %.2f hours\n", email, hoursLogged)

		if int(timeSpentSeconds/60/60) < tresholdHours {
			var att slack.AttachmentField
			att.Value = fmt.Sprintf("%s - %.2f hours logged\n", email, hoursLogged)

			attachment.Fields = append(attachment.Fields, att)
		}
	}

	_, _, _ = slackApi.PostMessage(os.Getenv("SLACK_CHANNEL"), slack.MsgOptionAttachments(attachment))
}

func retrieveWorklogIds() *WorklogIds {
	sinceDate := getSinceDate()
	sinceTs := sinceDate.UnixMilli()

	req, _ := jiraClient.NewRequest("GET", "/rest/api/3/worklog/updated?since="+strconv.FormatInt(sinceTs, 10), nil)

	worklog := &Worklog{}

	resRaw, err := jiraClient.Do(req, worklog)
	if err != nil {
		if Debug == "1" {
			bodyRaw, _ := io.ReadAll(resRaw.Body)
			spew.Dump(string(bodyRaw))
		}

		panic(err)
	}

	worklogIds := &WorklogIds{}

	for _, item := range worklog.ValueList {
		worklogIds.Ids = append(worklogIds.Ids, item.WorklogId)
	}

	return worklogIds
}

func retrieveWorklogInfoList(worklogIds *WorklogIds) *[]WorklogInfo {
	req, _ := jiraClient.NewRequest("POST", "/rest/api/3/worklog/list", worklogIds)

	worklogInfoList := &[]WorklogInfo{}

	resRaw, err := jiraClient.Do(req, worklogInfoList)

	if err != nil {
		if Debug == "1" {
			bodyRaw, _ := io.ReadAll(resRaw.Body)
			spew.Dump(string(bodyRaw))
		}

		panic(err)
	}

	return worklogInfoList
}

func calcSumWorkHoursEachUser(worklogInfoList *[]WorklogInfo, emailsForObservation Haystack) map[string]int64 {
	userWorklogList := make(map[string]int64)
	sinceDate := getSinceDate()

	for _, email := range emailsForObservation {
		userWorklogList[email] = 0
	}

	for _, item := range *worklogInfoList {
		started, _ := time.Parse("2006-01-02T15:04:05.999-0700", item.Started)

		if sinceDate.After(started) {
			continue
		}

		userWorklogList[item.Author.EmailAddress] += item.TimeSpentSeconds
	}

	return userWorklogList
}

func getSinceDate() time.Time {
	loc, _ := time.LoadLocation("Europe/Moscow")
	year, month, day := time.Now().In(loc).Date()

	return time.Date(year, month, day, 0, 0, 0, 0, loc)
}
