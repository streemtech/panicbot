package main

import (
	"fmt"
	"net/smtp"

	"github.com/bwmarrin/discordgo"
	"github.com/k0kubun/pp/v3"
	"github.com/twilio/twilio-go"
)

type Mail struct {
	Sender  string
	To      []string
	Cc      []string
	Bcc     []string
	Subject string
	Body    string
}

type Container struct {
	ConfigPath       string
	EmailAuth        smtp.Auth
	Session          *discordgo.Session
	TwilioRestClient *twilio.RestClient
}

func main() {
	fmt.Println("Hello Worlds")
	pp.Println("string")
}
