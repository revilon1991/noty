package main

import (
	"fmt"
	jira "github.com/andygrunwald/go-jira"
	"github.com/davecgh/go-spew/spew"
	"github.com/getlantern/systray"
	"github.com/joho/godotenv"
	"github.com/slack-go/slack"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var jiraClient *jira.Client
var slackClient *slack.Client
var emailsForObservation Haystack
var env map[string]string

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
var Version = "0"

func main() {
	Debug = "1"

	logSetup()
	env, _ = godotenv.Read()
	tp := jira.BasicAuthTransport{
		Username: env["JIRA_API_USERNAME"],
		Password: env["JIRA_API_TOKEN"],
	}
	jiraClient, _ = jira.NewClient(tp.Client(), env["JIRA_API_BASE_URL"])
	slackClient = slack.New(env["SLACK_TOKEN"])
	emailsForObservation = Haystack.Make(Haystack{}, env["JIRA_EMAILS_FOR_OBSERVATION"])

	systray.Run(onReady, nil)
}

func logSetup() {
	if Version != "0" {
		ep, err := os.Executable()
		if err != nil {
			log.Fatalln("os.Executable:", err)
		}
		err = os.Chdir(filepath.Join(filepath.Dir(ep), "..", "Resources"))
		if err != nil {
			log.Fatalln("os.Chdir:", err)
		}

		f, err := os.OpenFile("main.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		defer func() {
			err = f.Close()

			if err != nil {
				log.Panic(err)
			}
		}()

		err = syscall.Dup2(int(f.Fd()), int(os.Stderr.Fd()))
		if err != nil {
			log.Fatalf("Failed to redirect stderr to file: %v", err)
		}

		log.SetOutput(f)
	}
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
	worklogInfoList := &[]WorklogInfo{}

	if len(worklogIds.Ids) == 0 {
		return worklogInfoList
	}

	req, _ := jiraClient.NewRequest("POST", "/rest/api/3/worklog/list", worklogIds)

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

func onReady() {
	systray.SetIcon(icon)
	systray.SetTooltip("Noty")

	mAsk := systray.AddMenuItem("Ask logging", "Each one will get notify if threshold is over")
	systray.AddSeparator()

	mRefresh := systray.AddMenuItem("Refresh", "Show spent hours of every one")
	systray.AddSeparator()

	var employeeMenuList = make(map[string]*systray.MenuItem)

	for _, email := range emailsForObservation {
		employeeMenuList[email] = systray.AddMenuItem(fmt.Sprintf("%s - ? hours", email), "")
		employeeMenuList[email].Disable()
	}

	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the whole app")

	go func() {
		for {
			select {
			case <-mQuit.ClickedCh:
				systray.Quit()
			case <-mRefresh.ClickedCh:
				for email, employeeMenu := range employeeMenuList {
					employeeMenu.Disable()
					employeeMenuList[email].SetTitle(fmt.Sprintf("%s - ? hours", email))
				}

				worklogIds := retrieveWorklogIds()
				worklogInfoList := retrieveWorklogInfoList(worklogIds)
				sumWorkHoursEachUser := calcSumWorkHoursEachUser(worklogInfoList, emailsForObservation)

				for email, timeSpentSeconds := range sumWorkHoursEachUser {
					if !emailsForObservation.Has(email) {
						continue
					}

					hoursLogged := float64(timeSpentSeconds) / 60 / 60

					employeeMenuList[email].Enable()
					employeeMenuList[email].SetTitle(fmt.Sprintf("%s - %.2f hours", email, hoursLogged))
				}
			case <-mAsk.ClickedCh:
				thresholdHours, _ := strconv.Atoi(env["JIRA_THRESHOLD_HOURS"])

				attachment := slack.Attachment{
					Pretext: "Time log notification",
					Text:    "Do not forget log time for today",
					Color:   "#FFC700",
					Fields:  []slack.AttachmentField{},
				}

				worklogIds := retrieveWorklogIds()
				worklogInfoList := retrieveWorklogInfoList(worklogIds)
				sumWorkHoursEachUser := calcSumWorkHoursEachUser(worklogInfoList, emailsForObservation)

				for email, timeSpentSeconds := range sumWorkHoursEachUser {
					if !emailsForObservation.Has(email) {
						continue
					}

					hoursLogged := float64(timeSpentSeconds) / 60 / 60

					if int(timeSpentSeconds/60/60) < thresholdHours {
						var att slack.AttachmentField

						user, _ := slackClient.GetUserByEmail(email)

						if user == nil {
							att.Value = fmt.Sprintf("%s - %.2f hours logged\n", email, hoursLogged)
						} else {
							att.Value = fmt.Sprintf("<@%s> - %.2f hours logged\n", user.ID, hoursLogged)
						}

						attachment.Fields = append(attachment.Fields, att)
					}
				}

				_, _, _ = slackClient.PostMessage(env["SLACK_CHANNEL"], slack.MsgOptionAttachments(attachment))
			}
		}
	}()
}
