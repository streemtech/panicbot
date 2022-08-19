package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/streemtech/panicbot"
	"github.com/streemtech/panicbot/internal/slice"
	"sigs.k8s.io/yaml"
)

// TODO: Change debug logs to info.

// type Boot interface {
// 	configChanged() error
// 	configureLogger()
// 	watchFile(filePath string) error
// }

type AlertingMethods struct {
	Twilio Twilio
	Email  Email
}

type Config struct {
	DiscordBotToken  string
	GuildID          string
	PrimaryChannelID string
	AlertingMethods  AlertingMethods
	Voting           Voting
}

type ContactOnVote struct {
	Discord struct {
		Users []string
		Roles []string
	}
	Twilio struct {
		PhoneNumbers []string
	}
	Email struct {
		Addresses []string
	}
}

type Container struct {
	Config  Config
	Logger  *log.Logger
	Discord panicbot.Discord
	Twilio  panicbot.Twilio
}

type Email struct {
	Auth struct {
		Identity string
		Username string
		Password string
		Host     string
	}
	From           string
	DefaultMessage string
}
type RateLimit struct {
	PanicAlert struct {
		Day  int
		Hour int
	}
	PanicBan struct {
		Day  int
		Hour int
	}
}

type Twilio struct {
	AccountSID        string
	AuthToken         string
	TwilioPhoneNumber string
}

type Voting struct {
	AllowedToVote struct {
		PanicAlert struct {
			Users []string
			Roles []string
		}
		PanicBan struct {
			Users []string
			Roles []string
		}
	}
	Cooldown struct {
		PanicAlert string
		PanicBan   string
	}
	RequiredVotes struct {
		PanicAlert int
		PanicBan   int
	}
	VoteTimers struct {
		PanicAlertVoteTimer string
		PanicBanVoteTimer   string
	}
	ContactOnVote ContactOnVote
	RateLimit     RateLimit
}

func (c *Container) SendText() {
	for _, phoneNumber := range c.Config.Voting.ContactOnVote.Twilio.PhoneNumbers {
		c.Twilio.SendMessage(phoneNumber, c.Config.AlertingMethods.Twilio.TwilioPhoneNumber, "🚨Panic Ban Vote🚨\n\n Gami has triggered a Panic Ban vote against Xanzibaer.\n\n Reason: Blah\n\n Action Needed: Hop on Discord \n\n Ignore this message if you do not want to vote.")
	}
}

func (c *Container) PanicAlertCallback(message string) {
	// TODO write logic for starting a panicalert vote
	// TODO if enough votes then call SendDM method passing the information from the config.ContactOnVote {Discord {}} struct
	allUsers, err := c.Discord.GetAllGuildMembers()
	if err != nil {
		c.Logger.Errorf("failed to get all guild members")
	}
	for _, v := range allUsers {
		if compareVotePermissions(v.UserID, v.Roles, c.Config.Voting.AllowedToVote.PanicAlert.Users, c.Config.Voting.AllowedToVote.PanicAlert.Roles) {
			c.Discord.SendDM(v.UserID, message)
		}
	}
	// TODO if enough votes then call Twilio API to text/call the number from the config.ContactOnVote {Twilio {}} struct
	// TODO if enough votes then call Email handler to email the addresses from the config.ContactOnVote {Email {}} struct
	// TODO write logic for if vote fails. No one is contacted but perhaps a message is sent to the PrimaryChannel. Use SendChannelMessage
}

func (c *Container) PanicBanCallback(userID, targetUserID, reason string, days float64) {
	// TODO write logic for starting a panicban vote
	content := fmt.Sprintf("User <@%s> has triggered a Panic Ban vote against User <@%s>", userID, targetUserID)
	description := fmt.Sprintf("**Reason:** %s\n\n**Action Needed:** Click the Ban User button to cast your vote.\n\n**Ignore this message if you do not want to vote.**", reason)
	titleText := "🚨 Panic Ban Vote 🚨"
	buttonLabel := "Ban User"
	buttonID := uuid.New().String()
	allUsers, err := c.Discord.GetAllGuildMembers()
	if err != nil {
		c.Logger.Errorf("failed to get all guild members")
	}
	for _, v := range allUsers {
		if compareVotePermissions(v.UserID, v.Roles, c.Config.Voting.AllowedToVote.PanicAlert.Users, c.Config.Voting.AllowedToVote.PanicAlert.Roles) {
			err := c.Discord.SendDMEmbed(userID, content, description, titleText, buttonLabel, buttonID)
			if err != nil {
				c.Logger.Errorf("failed to send embeded direct message: %s", err.Error())
			}
		}
	}
	// TODO if enough votes then call Twilio API to text/call the number from the config.ContactOnVote {Twilio {}} struct
	// TODO if enough votes then call Email handler to email the addresses from the config.ContactOnVote {Email {}} struct
	// TODO if enough votes then call BanUser method
	// TODO write logic for if vote fails. No one is contacted but perhaps a message is sent to the PrimaryChannel. Use SendChannelMessage
}

func (c *Container) EmbedReactionCallback() {
	c.Logger.Info("Called!")
	// TODO use this for whenever we recieve a reaction to a panicalert / panicban
	// This function will be used to tally up the votes and then take action.
}

func compareVotePermissions(userID string, userRoles []string, allowedUserIDs []string, allowedUserRoles []string) bool {
	if slice.Contains(allowedUserIDs, userID) {
		return true
	}
	for _, userRole := range userRoles {
		if slice.Contains(allowedUserRoles, userRole) {
			return true
		}
	}
	return false
}

func main() {
	c := new(Container)
	c.configureLogger()
	err := c.configChanged(true)
	if err != nil {
		c.Logger.Fatalf("failed to load config: %s", err.Error())
	}
	err = c.startReloadRolesTimer()
	if err != nil {
		c.Logger.Fatalf("failed to start timer to check for update roles : %s", err.Error())
	}
	c.Discord, err = panicbot.NewDiscord(&panicbot.DiscordImplArgs{
		AllowedToVote:         c.Config.Voting.AllowedToVote,
		BotToken:              c.Config.DiscordBotToken,
		GuildID:               c.Config.GuildID,
		PrimaryChannelID:      c.Config.PrimaryChannelID,
		Logger:                c.Logger,
		EmbedReactionCallback: c.EmbedReactionCallback,
		PanicAlertCallback:    c.PanicAlertCallback,
		PanicBanCallback:      c.PanicBanCallback,
	})

	if err != nil {
		c.Logger.Fatalf("failed to create Discord session: %s", err)
	}
	c.Twilio, err = panicbot.NewTwilio(&panicbot.TwilioImplArgs{
		AuthToken:         c.Config.AlertingMethods.Twilio.AuthToken,
		AccountSID:        c.Config.AlertingMethods.Twilio.AccountSID,
		TwilioPhoneNumber: c.Config.AlertingMethods.Twilio.TwilioPhoneNumber,
		Logger:            c.Logger,
	})

	if err != nil {
		c.Logger.Fatalf("failed to create Twilio Rest Client: %s", err)
	}

	// err = c.watchFile("./config.yml")
	// if err != nil {
	// 	c.Logger.Fatalf("failed to watch configuration file: %s", err.Error())
	// }
	// Without the session I can't call Close()
	// defer c.Discord.Close()

	// pp.Println(c.Config)

	c.Logger.Debugf("create channel to listen for os interrupt")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	c.Logger.Infof("Press Ctrl+C to exit")
	<-stop

	c.Logger.Infof("Gracefully shutting down.")
	c.Discord.SendChannelMessage("", "So long!")
}

func (c *Container) configChanged(load bool) error {
	configFile := os.Getenv("CONFIG")
	if configFile == "" {
		configFile = "./config.yml"
		c.Logger.Infof("environment variable CONFIG was empty. Setting to default config file path location: %s", configFile)
	}
	yfile, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	conf := new(Config)

	err = yaml.Unmarshal(yfile, conf)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config data: %w", err)
	}
	// if load {
	err = c.loadConfig(*conf)
	// } else {
	// 	err = c.reloadConfig(*conf)
	// }
	if err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}
	return nil
}

// func (c *Container) watchFile(filePath string) error {
// 	// Code taken from: https://levelup.gitconnected.com/how-to-watch-for-file-change-in-golang-4d1eaa3d2964
// 	// Create a new file watcher.
// 	watcher, err := fsnotify.NewWatcher()
// 	if err != nil {
// 		return fmt.Errorf("failed to create config filewatcher: %w", err)
// 	}
// 	defer watcher.Close()
// 	_, err = os.Stat(filePath)
// 	if os.IsNotExist(err) {
// 		file, err := os.Create(filePath)
// 		if err != nil {
// 			return fmt.Errorf("failed to create file at filePath (%s) for filewatcher: %w", filePath, err)
// 		}
// 		file.Close()
// 	} else if err != nil {
// 		return fmt.Errorf("failed to stat file at filePath (%s) for filewatcher: %w", filePath, err)
// 	}
// 	err = watcher.Add(filePath)
// 	if err != nil {
// 		return fmt.Errorf("failed to add filePath (%s) to filewatcher: %w", filePath, err)
// 	}

// 	for {
// 		select {
// 		case event, ok := <-watcher.Events:
// 			if !ok {
// 				return fmt.Errorf("filewatcher events channel closed")
// 			}
// 			log.WithFields(log.Fields{
// 				"Name":      event.Name,
// 				"Operation": event.Op.String(),
// 			}).Debug("File event occurred")
// 			if event.Op == fsnotify.Write {
// 				err = c.configChanged(false)
// 				if err != nil {
// 					return fmt.Errorf("failed to update config: %w", err)
// 				}
// 			}
// 		case err, ok := <-watcher.Errors:
// 			if !ok {
// 				return fmt.Errorf("filewatcher errors channel closed")
// 			}
// 			return fmt.Errorf("filewatcher error encountered: %w", err)
// 		}
// 	}
// }

func (c *Container) configureLogger() {
	c.Logger = log.StandardLogger()
	c.Logger.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})
	c.Logger.SetOutput(os.Stdout)
	level := os.Getenv("LOGLEVEL")
	if level == "" {
		level = "info"
	}
	logLevel, err := log.ParseLevel(level)
	if err != nil {
		logLevel = log.InfoLevel
		c.Logger.Errorf("unable to parse log level %s:%s", level, err.Error())
	}
	c.Logger.SetLevel(logLevel)
	c.Logger.Infof("log level set to: %s", logLevel.String())
}
