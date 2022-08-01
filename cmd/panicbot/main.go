package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/k0kubun/pp/v3"
	"github.com/sirupsen/logrus"
	"github.com/twilio/twilio-go"
)

type Config struct {
	ConfigPath string
	Mail       struct {
		Sender  string
		To      []string
		CC      []string
		BCC     []string
		Subject string
		Body    string
		Addr    string // Mail server address with port number.
		Auth    struct {
			Username string // Email address to be used for auth.
			Hostname string // Mail server address.
			Password string
		}
	}
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
