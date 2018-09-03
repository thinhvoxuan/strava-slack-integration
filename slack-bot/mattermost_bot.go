// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"time"

	"github.com/mattermost/mattermost-server/model"
	"github.com/shomali11/proper"
)

const (
	BOT_NAME  = "Mr Kofi Bot"
	ENDPOINTS = "mattermost.geekup.vn"

	USER_EMAIL    = "mr_kofi_bot@geekup.vn"
	USER_PASSWORD = "Password1"
	USER_NAME     = "mr_kofi"
	USER_FIRST    = "Kofi"
	USER_LAST     = "Bot"

	TEAM_NAME        = "geek-up"
	CHANNEL_LOG_NAME = "mr_kofi_bot_channel"
)

var client *model.Client4
var webSocketClient *model.WebSocketClient

var botUser *model.User
var botTeam *model.Team
var debuggingChannel *model.Channel

var listCommand = []BotCommand{}

// Documentation for the Go driver can be found
// at https://godoc.org/github.com/mattermost/platform/model#Client
func InitMatterMostBot() {
	println(BOT_NAME)

	SetupGracefulShutdown()

	client = model.NewAPIv4Client("https://" + ENDPOINTS)

	// Lets test to see if the mattermost server is up and running
	MakeSureServerIsRunning()

	// lets attempt to login to the Mattermost server as the bot user
	// This will set the token required for all future calls
	// You can get this token with client.AuthToken
	LoginAsTheBotUser()

	// If the bot user doesn't have the correct information lets update his profile
	UpdateTheBotUserIfNeeded()

	// Lets find our bot team
	FindBotTeam()

	// This is an important step.  Lets make sure we use the botTeam
	// for all future web service requests that require a team.
	//client.SetTeamId(botTeam.Id)

	// Lets create a bot channel for logging debug messages into
	CreateBotDebuggingChannelIfNeeded()
	println("Completed starting")
	SendMsg("_"+BOT_NAME+" has **started** running_", "", "")
	SetupCommandMessage()

	// Lets start listening to some channels via the websocket!
	webSocketClient, err := model.NewWebSocketClient4("wss://"+ENDPOINTS, client.AuthToken)
	if err != nil {
		println("We failed to connect to the web socket")
		PrintError(err)
	}

	webSocketClient.Listen()

	go func() {
		for {
			select {
			case resp := <-webSocketClient.EventChannel:
				HandleWebSocketResponse(resp)
			}
		}
	}()

	// You can block forever with
	select {}
}

func MakeSureServerIsRunning() {
	if props, resp := client.GetOldClientConfig(""); resp.Error != nil {
		println("There was a problem pinging the Mattermost server.  Are you sure it's running?")
		PrintError(resp.Error)
		os.Exit(1)
	} else {
		println("Server detected and is running version " + props["Version"])
	}
}

func LoginAsTheBotUser() {
	if user, resp := client.Login(USER_EMAIL, USER_PASSWORD); resp.Error != nil {
		println("There was a problem logging into the Mattermost server.  Are you sure ran the setup steps from the README.md?")
		PrintError(resp.Error)
		os.Exit(1)
	} else {
		botUser = user
	}
}

func UpdateTheBotUserIfNeeded() {
	if botUser.FirstName != USER_FIRST || botUser.LastName != USER_LAST || botUser.Username != USER_NAME {
		botUser.FirstName = USER_FIRST
		botUser.LastName = USER_LAST
		botUser.Username = USER_NAME

		if user, resp := client.UpdateUser(botUser); resp.Error != nil {
			println("We failed to update the Sample Bot user")
			PrintError(resp.Error)
			os.Exit(1)
		} else {
			botUser = user
			println("Looks like this might be the first run so we've updated the bots account settings")
		}
	}
}

func FindBotTeam() {
	if team, resp := client.GetTeamByName(TEAM_NAME, ""); resp.Error != nil {
		println("We failed to get the initial load")
		println("or we do not appear to be a member of the team '" + TEAM_NAME + "'")
		PrintError(resp.Error)
		os.Exit(1)
	} else {
		botTeam = team
	}
}

func CreateBotDebuggingChannelIfNeeded() {
	if rchannel, resp := client.GetChannelByName(CHANNEL_LOG_NAME, botTeam.Id, ""); resp.Error != nil {
		println("We failed to get the channels")
		PrintError(resp.Error)
	} else {
		debuggingChannel = rchannel
		return
	}

	// Looks like we need to create the logging channel
	channel := &model.Channel{}
	channel.Name = CHANNEL_LOG_NAME
	channel.DisplayName = "Mr Kofi bot channel"
	channel.Purpose = "Mr Kofi bot channel bot debug messages"
	channel.Type = model.CHANNEL_OPEN
	channel.TeamId = botTeam.Id
	if rchannel, resp := client.CreateChannel(channel); resp.Error != nil {
		println("We failed to create the channel " + CHANNEL_LOG_NAME)
		PrintError(resp.Error)
	} else {
		debuggingChannel = rchannel
		println("Looks like this might be the first run so we've created the channel " + CHANNEL_LOG_NAME)
	}
}

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
	case "lastweek":
		fromDate = startOfWeek().AddDate(0, 0, -7).Format(TIMEFORMAT)
		toDate = startOfWeek().Format(TIMEFORMAT)
	case "this-week":
	case "thisweek":
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

func SetupCommandMessage() {
	cmd1 := NewBotCommand("Log summary <project_alias> <time>", "Show the log summary for project `project_alias` `last-week|this-week|yesterday|today|dd-mm-yyyy~dd-mm-yyyy`", func(textInput string, parameters *proper.Properties) string {
		projectAlias := parameters.StringParam("project_alias", "")
		time := parameters.StringParam("time", "today")
		fromDate, toDate := parseTime(time)
		result := WorklogSummary(fromDate, toDate, projectAlias)
		return result
	})
	cmd2 := NewBotCommand("Log detail <project_alias> <time>", "Show the detail logwork with message for `project_alias` `last-week|this-week|yesterday|today|dd-mm-yyyy~dd-mm-yyyy`", func(textInput string, parameters *proper.Properties) string {
		projectAlias := parameters.StringParam("project_alias", "")
		time := parameters.StringParam("time", "today")
		fromDate, toDate := parseTime(time)
		result := WorklogDetail(fromDate, toDate, projectAlias)
		return result
	})
	cmd3 := NewBotCommand("help", "help", func(textInput string, parameters *proper.Properties) string {
		helpMessage := ""
		for _, command := range listCommand {
			tokens := command.Tokenize()
			for _, token := range tokens {
				if token.IsParameter {
					helpMessage += fmt.Sprintf("`%s`", token.Word) + " "
				} else {
					helpMessage += fmt.Sprintf("*%s*", token.Word) + " "
				}
			}
			helpMessage += "- " + fmt.Sprintf("_%s_", command.Description()) + "\n"
		}
		return helpMessage
	})
	listCommand = append(listCommand, cmd1)
	listCommand = append(listCommand, cmd2)
	listCommand = append(listCommand, cmd3)
}

func SendMsg(msg string, replyToId string, channelId string) {
	post := &model.Post{}
	if len(channelId) == 0 {
		channelId = debuggingChannel.Id
	}
	post.ChannelId = channelId
	post.Message = msg

	post.RootId = replyToId

	if _, resp := client.CreatePost(post); resp.Error != nil {
		println("We failed to send a message to the logging channel")
		PrintError(resp.Error)
	}
}

func HandleWebSocketResponse(event *model.WebSocketEvent) {
	HandleMsg(event)
}

func HandleMsg(event *model.WebSocketEvent) {
	// If this isn't the debugging channel then lets ingore it
	// if event.Broadcast.ChannelId != debuggingChannel.Id {
	// 	return
	// }

	// Lets only reponded to messaged posted events
	if event.Event != model.WEBSOCKET_EVENT_POSTED {
		return
	}

	println("New message is comming")

	post := model.PostFromJson(strings.NewReader(event.Data["post"].(string)))
	if post != nil {

		// ignore my events
		if post.UserId == botUser.Id {
			return
		}

		// ignore if not call me
		if matched, _ := regexp.MatchString(`(?:^|\W)mr_kofi(?:$|\W)`, post.Message); !matched {
			return
		}

		// Trim space and new line
		post.Message = strings.TrimSpace(post.Message)

		println("-s-" + post.Message + "-s-")

		for _, cmd := range listCommand {
			parameters, isMatch := cmd.Match(post.Message)
			if !isMatch {
				continue
			}

			reply := cmd.Execute(post.Message, parameters)
			SendMsg(reply, post.Id, post.ChannelId)
			return
		}
	}

	SendMsg("I don't understand you. Please type `How can you hellp me @mr_kofi ?`", post.Id, post.ChannelId)
}

func PrintError(err *model.AppError) {
	println("\tError Details:")
	println("\t\t" + err.Message)
	println("\t\t" + err.Id)
	println("\t\t" + err.DetailedError)
}

func SetupGracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			if webSocketClient != nil {
				webSocketClient.Close()
			}

			SendMsg("_"+BOT_NAME+" has **stopped** running_", "", "")
			os.Exit(0)
		}
	}()
}
