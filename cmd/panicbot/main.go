package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/k0kubun/pp/v3"
	"github.com/sirupsen/logrus"
	"github.com/twilio/twilio-go"
)

type Config struct {
	// TODO: Put config here.
}

type Container struct {
	Config           Config
	Logger           *logrus.Logger
	Session          *discordgo.Session
	TwilioRestClient *twilio.RestClient
}

func main() {
	fmt.Println("Hello Worlds")
	pp.Println("string")
}
