package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/shomali11/slacker"
)

func handleError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

const TIMEFORMAT = "02-01-2006"

func startOfWeek() time.Time {
	now := time.Now()
	for now.Weekday() != time.Saturday {
		now = now.AddDate(0, 0, -1)
	}
	return now
}

func parseTime(inputTime string) (fromDate string, toDate string) {
	fromDate = time.Now().Format(TIMEFORMAT)
	toDate = time.Now().AddDate(0, 0, 1).Format(TIMEFORMAT)

	switch inputTime {
	case "yesterday":
		fromDate = time.Now().AddDate(0, 0, -1).Format(TIMEFORMAT)
		toDate = time.Now().Format(TIMEFORMAT)
	case "last-week":
		fromDate = startOfWeek().AddDate(0, 0, -7).Format(TIMEFORMAT)
		toDate = startOfWeek().Format(TIMEFORMAT)
	case "this-week":
		fromDate = startOfWeek().Format(TIMEFORMAT)
		toDate = startOfWeek().AddDate(0, 0, 7).Format(TIMEFORMAT)
	default:
		dateRange := strings.Split(inputTime, "~")
		if len(dateRange) == 2 {
			fromDate = dateRange[0]
			toDate = dateRange[1]
		}
	}
	return
}

func initBot() {
	godotenv.Load()
	bot := slacker.NewClient(os.Getenv("TOKEN"))

	bot.Command("Worklog <project_alias> <time>", "Worklog `hyperloop` `last-week|this-week|yesterday|today|dd-mm-yyyy~dd-mm-yyyy`", func(request slacker.Request, response slacker.ResponseWriter) {
		response.Typing()
		projectAlias := request.StringParam("project_alias", "")
		time := request.StringParam("time", "today")
		fromDate, toDate := parseTime(time)
		result := WorklogSummary(fromDate, toDate, projectAlias)
		response.Reply(result)
	})

	bot.Command("Log detail <project_alias> <time>", "Worklog `hyperloop` `last-week|this-week|yesterday|today|dd-mm-yyyy~dd-mm-yyyy`", func(request slacker.Request, response slacker.ResponseWriter) {
		response.Typing()
		projectAlias := request.StringParam("project_alias", "")
		time := request.StringParam("time", "today")
		fromDate, toDate := parseTime(time)
		result := WorklogDetail(fromDate, toDate, projectAlias)
		response.Reply(result)
	})

	log.Println("Complete loading")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errListen := bot.Listen(ctx)
	if errListen != nil {
		log.Fatal(errListen)
	}
}

func main() {
	initBot()
}
