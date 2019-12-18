package main

import (
	"github.com/shadeyrl/bot"
	"time"
)

func main() {
	myBot := bot.BasicBot{
		Channel: "twitch",
		MsgRate: time.Duration(20/30) * time.Millisecond,
		Name: "lorp",
		Port: "6667",
		PrivatePath: "../private/oauth.json",
		Server: "irc.chat.twitch.tv"
	}
	myBot.Start()
}