// Copyright (c) 2016 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package samplebot

import (
	"os"
	"os/signal"
	"regexp"
	"strings"

	"github.com/mattermost/mattermost-server/v5/model"
)

//Constant values
const (
	SampleName = "Mattermost Bot Sample"

	UserEmail    = "bot@example.com"
	UserPassword = "1qaz@WSX3edc"
	UserName     = "samplebot"
	UserFirst    = "Sample"
	UserLast     = "Bot"

	TeamName       = "localteam"
	ChannelLogName = "debugging-for-sample-bot"
)

var client *model.Client4
var webSocketClient *model.WebSocketClient

var botUser *model.User
var botTeam *model.Team
var debuggingChannel *model.Channel

// Documentation for the Go driver can be found
// at https://godoc.org/github.com/mattermost/platform/model#Client
func main() {
	println(SampleName)

	setupGracefulShutdown()

	client = model.NewAPIv4Client("http://localhost:8065")

	// Lets test to see if the mattermost server is up and running
	makeSureServerIsRunning()

	// lets attempt to login to the Mattermost server as the bot user
	// This will set the token required for all future calls
	// You can get this token with client.AuthToken
	loginAsTheBotUser()

	// If the bot user doesn't have the correct information lets update his profile
	updateTheBotUserIfNeeded()

	// Lets find our bot team
	findBotTeam()

	// This is an important step.  Lets make sure we use the botTeam
	// for all future web service requests that require a team.
	//client.SetTeamId(botTeam.Id)

	// Lets create a bot channel for logging debug messages into
	createBotDebuggingChannelIfNeeded()
	sendMsgToDebuggingChannel("_"+SampleName+" has **started** running_", "")

	// Lets start listening to some channels via the websocket!
	webSocketClient, err := model.NewWebSocketClient4("ws://localhost:8065", client.AuthToken)
	if err != nil {
		println("We failed to connect to the web socket")
		printError(err)
	}

	webSocketClient.Listen()

	go func() {
		for {
			select {
			case resp := <-webSocketClient.EventChannel:
				handleWebSocketResponse(resp)
			}
		}
	}()

	// You can block forever with
	select {}
}

func makeSureServerIsRunning() {
	if props, resp := client.GetOldClientConfig(""); resp.Error != nil {
		println("There was a problem pinging the Mattermost server.  Are you sure it's running?")
		printError(resp.Error)
		os.Exit(1)
	} else {
		println("Server detected and is running version " + props["Version"])
	}
}

func loginAsTheBotUser() {
	if user, resp := client.Login(UserEmail, UserPassword); resp.Error != nil {
		println("There was a problem logging into the Mattermost server.  Are you sure ran the setup steps from the README.md?")
		printError(resp.Error)
		os.Exit(1)
	} else {
		botUser = user
	}
}

func updateTheBotUserIfNeeded() {
	if botUser.FirstName != UserFirst || botUser.LastName != UserLast || botUser.Username != UserName {
		botUser.FirstName = UserFirst
		botUser.LastName = UserLast
		botUser.Username = UserName

		if user, resp := client.UpdateUser(botUser); resp.Error != nil {
			println("We failed to update the Sample Bot user")
			printError(resp.Error)
			os.Exit(1)
		} else {
			botUser = user
			println("Looks like this might be the first run so we've updated the bots account settings")
		}
	}
}

func findBotTeam() {
	if team, resp := client.GetTeamByName(TeamName, ""); resp.Error != nil {
		println("We failed to get the initial load")
		println("or we do not appear to be a member of the team '" + TeamName + "'")
		printError(resp.Error)
		os.Exit(1)
	} else {
		botTeam = team
	}
}

func createBotDebuggingChannelIfNeeded() {
	if rchannel, resp := client.GetChannelByName(ChannelLogName, botTeam.Id, ""); resp.Error != nil {
		println("We failed to get the channels")
		printError(resp.Error)
	} else {
		debuggingChannel = rchannel
		return
	}

	// Looks like we need to create the logging channel
	channel := &model.Channel{}
	channel.Name = ChannelLogName
	channel.DisplayName = "Debugging For Sample Bot"
	channel.Purpose = "This is used as a test channel for logging bot debug messages"
	channel.Type = model.CHANNEL_OPEN
	channel.TeamId = botTeam.Id
	if rchannel, resp := client.CreateChannel(channel); resp.Error != nil {
		println("We failed to create the channel " + ChannelLogName)
		printError(resp.Error)
	} else {
		debuggingChannel = rchannel
		println("Looks like this might be the first run so we've created the channel " + ChannelLogName)
	}
}

func sendMsgToDebuggingChannel(msg string, replyToID string) {
	post := &model.Post{}
	post.ChannelId = debuggingChannel.Id
	post.Message = msg

	post.RootId = replyToID

	if _, resp := client.CreatePost(post); resp.Error != nil {
		println("We failed to send a message to the logging channel")
		printError(resp.Error)
	}
}

func handleWebSocketResponse(event *model.WebSocketEvent) {
	handleMsgFromDebuggingChannel(event)
}

func handleMsgFromDebuggingChannel(event *model.WebSocketEvent) {
	// If this isn't the debugging channel then lets ingore it
	if event.Broadcast.ChannelId != debuggingChannel.Id {
		return
	}

	// Lets only reponded to messaged posted events
	if event.Event != model.WEBSOCKET_EVENT_POSTED {
		return
	}

	println("responding to debugging channel msg")

	post := model.PostFromJson(strings.NewReader(event.Data["post"].(string)))
	if post != nil {

		// ignore my events
		if post.UserId == botUser.Id {
			return
		}

		// if you see any word matching 'alive' then respond
		if matched, _ := regexp.MatchString(`(?:^|\W)alive(?:$|\W)`, post.Message); matched {
			sendMsgToDebuggingChannel("Yes I'm running", post.Id)
			return
		}

		// if you see any word matching 'up' then respond
		if matched, _ := regexp.MatchString(`(?:^|\W)up(?:$|\W)`, post.Message); matched {
			sendMsgToDebuggingChannel("Yes I'm running", post.Id)
			return
		}

		// if you see any word matching 'running' then respond
		if matched, _ := regexp.MatchString(`(?:^|\W)running(?:$|\W)`, post.Message); matched {
			sendMsgToDebuggingChannel("Yes I'm running", post.Id)
			return
		}

		// if you see any word matching 'hello' then respond
		if matched, _ := regexp.MatchString(`(?:^|\W)hello(?:$|\W)`, post.Message); matched {
			sendMsgToDebuggingChannel("Yes I'm running", post.Id)
			return
		}
	}

	sendMsgToDebuggingChannel("I did not understand you!", post.Id)
}

func printError(err *model.AppError) {
	println("\tError Details:")
	println("\t\t" + err.Message)
	println("\t\t" + err.Id)
	println("\t\t" + err.DetailedError)
}

func setupGracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			if webSocketClient != nil {
				webSocketClient.Close()
			}

			sendMsgToDebuggingChannel("_"+SampleName+" has **stopped** running_", "")
			os.Exit(0)
		}
	}()
}
