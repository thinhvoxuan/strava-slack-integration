package main

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/shomali11/slacker"
)

func handleError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

const TIMEFORMAT = "02-01-2006"

func initSlackBot() {
	godotenv.Load()
	bot := slacker.NewClient(os.Getenv("TOKEN"))

	bot.Command("Log summary <project_alias> <time>", "Worklog `hyperloop` `last-week|this-week|yesterday|today|dd-mm-yyyy~dd-mm-yyyy`", func(request slacker.Request, response slacker.ResponseWriter) {
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
	// initSlackBot()
	InitMatterMostBot()
}
