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
	// Workload  []Workload
}

func (user UserLogWork) totalWorklog() (sum float64) {
	sum = 0
	for _, log := range user.Worklog {
		sum += log.Hours
	}
	return sum
}

func (user UserLogWork) groupLogworkByproject() (sProjectLog map[string]UserLogWork) {
	sProjectLog = map[string]UserLogWork{}
	for _, log := range user.Worklog {
		projectName := log.ProjectName
		userLog, ok := sProjectLog[projectName]
		if !ok {
			userLog = UserLogWork{
				UserAlias: user.UserAlias,
			}
		}
		userLog.Worklog = append(userLog.Worklog, log)
		sProjectLog[projectName] = userLog
	}
	return
}

func (user UserLogWork) summaryLogByProject() (result string) {
	result = "@" + user.UserAlias + "\n"
	sProjectLog := user.groupLogworkByproject()
	for projectName, userLog := range sProjectLog {
		result += fmt.Sprintf("\t+ %s: %.2fh\n", projectName, userLog.totalWorklog())
	}
	result += "\n"
	return
}

func (user UserLogWork) detailLogByProject() (result string) {
	result = "@" + user.UserAlias + "\n"
	sProjectLog := user.groupLogworkByproject()
	for projectName, userLog := range sProjectLog {
		result += fmt.Sprintf("  + *%s:* %.2fh\n", projectName, userLog.totalWorklog())
		for _, log := range userLog.Worklog {
			result += fmt.Sprintf("\t- %.2f: %s\n", log.Hours, log.LogMessage)
		}
	}
	result += "\n"
	return
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
// type Attendance struct {
// 	UserAlias      string `json:"user_alias"`
// 	AttendanceType string `json:"attendance_type"`
// 	Weight         int    `json:"weight"`
// 	ForDay         string `json:"for_day"`
// 	ReportedDate   string `json:"reported_date"`
// 	Comment        string `json:"comment"`
// }

// Workload record
// type Workload struct {
// 	ProjectAlias string  `json:"project_alias"`
// 	ProjectName  string  `json:"project_name"`
// 	MemberAlias  string  `json:"member_alias"`
// 	CommitTime   float64 `json:"commit_time"`
// 	Week         string  `json:"week"`
// 	WeekName     string  `json:"week_name"`
// 	IsApproved   int     `json:"is_approved"`
// }

// ExportData record
type ExportData struct {
	Worklog []WorkLog `json:"worklog"`
	// Attendance []Attendance `json:"attendance"`
	// Workload   []Workload   `json:"workload"`
}

func fetchInternalInformation(fromDate string, toDate string, projectAlias string) (export ExportData) {
	resp, err := resty.R().SetQueryParams(map[string]string{
		"start":         fromDate,
		"stop":          toDate,
		"project_alias": projectAlias,
	}).Get("https://internal.geekup.vn/api/export/worklog")

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
	export := fetchInternalInformation(fromDate, toDate, projectAlias)
	sUser := summaryByMember(export, projectAlias)
	summary = fmt.Sprintf("Project `%s` (`%s` to `%s`) summary:\n", projectAlias, fromDate, toDate)
	for _, user := range sUser {
		summary += user.summaryLogByProject()
	}
	return
}

//WorklogDetail export logwork from date - to date
func WorklogDetail(fromDate string, toDate string, projectAlias string) (summary string) {
	export := fetchInternalInformation(fromDate, toDate, projectAlias)
	sUser := summaryByMember(export, projectAlias)
	summary = fmt.Sprintf("Detail Project log `%s` (`%s` to `%s`):\n", projectAlias, fromDate, toDate)
	for _, user := range sUser {
		summary += user.detailLogByProject()
	}
	return
}
