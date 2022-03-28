package main

import (
	jira "github.com/andygrunwald/go-jira"
	"github.com/davecgh/go-spew/spew"
	"github.com/joho/godotenv"
	"os"
	"strconv"
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
		DisplayName string `json:"displayName"`
		AccountId   string `json:"accountId"`
		Active      bool   `json:"active"`
	} `json:"author"`
	UpdateAuthor struct {
		DisplayName string `json:"displayName"`
		AccountId   string `json:"accountId"`
		Active      bool   `json:"active"`
	} `json:"updateAuthor"`
	TimeSpent        string `json:"timeSpent"`
	TimeSpentSeconds int64  `json:"timeSpentSeconds"`
}

func main() {
	_ = godotenv.Load()

	tp := jira.BasicAuthTransport{
		Username: os.Getenv("JIRA_API_USERNAME"),
		Password: os.Getenv("JIRA_API_TOKEN"),
	}

	jiraClient, _ = jira.NewClient(tp.Client(), os.Getenv("JIRA_API_BASE_URL"))

	worklogIds := retrieveWorklogIds()
	worklogInfoList := retrieveWorklogInfoList(worklogIds)

	spew.Dump(worklogInfoList)
}

func retrieveWorklogIds() *WorklogIds {
	now := time.Now()
	year, month, day := now.Date()
	since := time.Date(year, month, day, 0, 0, 0, 0, now.Location()).UnixMilli()

	req, _ := jiraClient.NewRequest("GET", "/rest/api/3/worklog/updated?since="+strconv.FormatInt(since, 10), nil)

	worklog := &Worklog{}

	_, err := jiraClient.Do(req, worklog)
	if err != nil {
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

	_, err := jiraClient.Do(req, worklogInfoList)

	if err != nil {
		panic(err)
	}

	return worklogInfoList
}
