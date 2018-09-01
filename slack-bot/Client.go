package main

import (
	"encoding/json"
	"fmt"
	"strings"

	resty "gopkg.in/resty.v1"
)

type UserLogWork struct {
	UserAlias string
	Worklog   []WorkLog
	Workload  []Workload
}

func (user UserLogWork) totalWorklog() (sum float64) {
	sum = 0
	for _, log := range user.Worklog {
		sum += log.Hours
	}
	return sum
}

// WorkLog record
type WorkLog struct {
	ProjectAlias string  `json:"project_alias"`
	ProjectName  string  `json:"project_name"`
	MemberAlias  string  `json:"member_alias"`
	WorkUnit     string  `json:"work_unit"`
	LogType      string  `json:"log_type"`
	LogForDate   string  `json:"log_for_date"`
	Hours        float64 `json:"hours"`
	LogMessage   string  `json:"log_message"`
	LoggedOn     string  `json:"logged_on"`
}

// Attendance record
type Attendance struct {
	UserAlias      string `json:"user_alias"`
	AttendanceType string `json:"attendance_type"`
	Weight         int    `json:"weight"`
	ForDay         string `json:"for_day"`
	ReportedDate   string `json:"reported_date"`
	Comment        string `json:"comment"`
}

// Workload record
type Workload struct {
	ProjectAlias string  `json:"project_alias"`
	ProjectName  string  `json:"project_name"`
	MemberAlias  string  `json:"member_alias"`
	CommitTime   float64 `json:"commit_time"`
	Week         string  `json:"week"`
	WeekName     string  `json:"week_name"`
	IsApproved   int     `json:"is_approved"`
}

// ExportData record
type ExportData struct {
	Worklog    []WorkLog    `json:"worklog"`
	Attendance []Attendance `json:"attendance"`
	Workload   []Workload   `json:"workload"`
}

func fetchInternalInformation(fromDate string, toDate string) (export ExportData) {
	resp, err := resty.R().SetQueryParams(map[string]string{
		"start": fromDate,
		"stop":  toDate,
	}).Get("https://internal.geekup.vn/api/export/data")

	handleError(err)
	json.Unmarshal(resp.Body(), &export)
	return
}

func summaryByMember(export ExportData, projectAlias string) (sUser map[string]UserLogWork) {
	sUser = map[string]UserLogWork{}
	for _, workRecord := range export.Worklog {
		matching := strings.Contains(workRecord.ProjectAlias, projectAlias) ||
			strings.Contains(projectAlias, workRecord.ProjectAlias)
		if !matching {
			continue
		}

		memberAlias := workRecord.MemberAlias
		user, ok := sUser[memberAlias]
		if !ok {
			user = UserLogWork{
				UserAlias: memberAlias,
			}
		}
		user.Worklog = append(user.Worklog, workRecord)
		sUser[memberAlias] = user
	}
	return
}

//WorklogSummary export logwork from date - to date
func WorklogSummary(fromDate string, toDate string, projectAlias string) (summary string) {
	export := fetchInternalInformation(fromDate, toDate)
	sUser := summaryByMember(export, projectAlias)
	summary = fmt.Sprintf("Project `%s` (`%s` to `%s`) summary:\n", projectAlias, fromDate, toDate)
	for _, user := range sUser {
		summary += fmt.Sprintf("\t+ @%s: %.2fh\n", user.UserAlias, user.totalWorklog())
	}
	return
}

//WorklogDetail export logwork from date - to date
func WorklogDetail(fromDate string, toDate string, projectAlias string) (summary string) {
	export := fetchInternalInformation(fromDate, toDate)
	sUser := summaryByMember(export, projectAlias)
	summary = fmt.Sprintf("Detail Project log `%s` (`%s` to `%s`):\n", projectAlias, fromDate, toDate)
	for _, user := range sUser {
		summary += fmt.Sprintf("*@%s:* %.2fh\n", user.UserAlias, user.totalWorklog())
		for _, log := range user.Worklog {
			summary += fmt.Sprintf("\t+ %.2fh: @%s\n", log.Hours, log.LogMessage)
		}
		summary += fmt.Sprintf("\n")
	}
	return
}
