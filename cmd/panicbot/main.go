package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/streemtech/panicbot"
	"github.com/streemtech/panicbot/internal/slice"
	"sigs.k8s.io/yaml"
)

const PANIC_BAN_VOTE_TYPE = "panicban"
const PANIC_ALERT_VOTE_TYPE = "panicalert"

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
	Config      Config
	Logger      *log.Logger
	Discord     panicbot.Discord
	Twilio      panicbot.Twilio
	GracePeriod map[string]time.Time
	VoteTracker map[string]VoteData
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
	APIKey            string
	APISecret         string
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

type VoteData struct {
	AlertMessage string
	CallingUser  string
	PanicType    string
	Voters       map[string]bool

	// Optional, only for Ban
	Days       float64
	BanReason  string
	TargetUser string
}

func (c *Container) SendText(message string) {
	for _, phoneNumber := range c.Config.Voting.ContactOnVote.Twilio.PhoneNumbers {
		c.Twilio.SendMessage(phoneNumber, message)
	}
}

func (c *Container) PanicAlertCallback(message string) {
	// TODO write logic for starting a panicalert vote
	// TODO if enough votes then call SendDM method passing the information from the config.ContactOnVote {Discord {}} struct
	allUsers, err := c.Discord.GetAllGuildMembers()
	if err != nil {
		c.Logger.Errorf("failed to get all guild members: %s", err.Error())
	}
	for _, v := range allUsers {
		if hasVotePermissions(v.UserID, v.Roles, c.Config.Voting.AllowedToVote.PanicAlert.Users, c.Config.Voting.AllowedToVote.PanicAlert.Roles) || c.RoleRemovedCheck(v.UserID) {
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

	voteID := uuid.New().String()

	c.VoteTracker[voteID] = VoteData{
		Voters:      make(map[string]bool),
		CallingUser: userID,
		PanicType:   PANIC_BAN_VOTE_TYPE,
		Days:        days,
		BanReason:   reason,
		TargetUser:  targetUserID,
	}
	allUsers, err := c.Discord.GetAllGuildMembers()
	if err != nil {
		c.Logger.Errorf("failed to get all guild members: %s", err.Error())
	}
	for _, v := range allUsers {
		if hasVotePermissions(v.UserID, v.Roles, c.Config.Voting.AllowedToVote.PanicBan.Users, c.Config.Voting.AllowedToVote.PanicBan.Roles) {
			err := c.Discord.SendDMEmbed(userID, content, description, titleText, buttonLabel, voteID)
			if err != nil {
				c.Logger.Errorf("failed to send embedded direct message: %s", err.Error())
			}
		}
	}

	voteTime, err := time.ParseDuration(c.Config.Voting.VoteTimers.PanicBanVoteTimer)
	if err != nil {
		c.Logger.Errorf("failed to parse ban vote duration: %s ,setting to default time of five minutes", err.Error())
		voteTime = time.Minute * 5
	}
	go time.AfterFunc(voteTime, func() {
		voteData, ok := c.VoteTracker[voteID]
		if !ok {
			return
		}
		// Remove the vote from VoteTracker. The vote failed(Not enough people voted to ban.)
		delete(c.VoteTracker, voteID)

		member, err := c.Discord.GetGuildMemberUsername(voteData.TargetUser)
		if err != nil {
			c.Logger.Errorf("failed to get GuildMember: %s", err.Error())
		}
		// Send message saying that the vote failed.
		c.Discord.SendChannelMessage("", fmt.Sprintf("Vote to ban user %s has failed. Time elapsed and not enough votes received", member))
	})
}

func (c *Container) RoleRemovedCallback(user string, role string) {
	if !hasVotePermissions("", []string{role}, []string{}, c.Config.Voting.AllowedToVote.PanicBan.Roles) {
		return
	}
	t := time.Now()
	c.GracePeriod[user] = t
	time.AfterFunc(time.Minute*30, func() {
		t2 := c.GracePeriod[user]
		if t == t2 {
			delete(c.GracePeriod, user)
		}
	})
}
func (c *Container) RoleRemovedCheck(user string) bool {
	_, ok := c.GracePeriod[user]
	return ok
}

func (c *Container) EmbedReactionCallback(userID, voteID string) {
	voteData, ok := c.VoteTracker[voteID]
	if !ok {
		err := c.Discord.SendDM(userID, "Sorry, this vote has ended")
		if err != nil {
			c.Logger.Errorf("could not notify the user that the vote ended: %s", err.Error())
		}
		return
	}
	switch voteData.PanicType {
	case PANIC_ALERT_VOTE_TYPE:
		// TODO: Panic Alert stuff here when they click the button
	case PANIC_BAN_VOTE_TYPE:
		// Check to see if the voter is already in the voters array.
		_, ok := voteData.Voters[userID]
		if ok {
			err := c.Discord.SendDM(userID, "Sorry, you have already participated in this vote")
			if err != nil {
				c.Logger.Errorf("failed to send DM: %s", err.Error())
			}
			return
		}
		// Add the user to the Voters array and let them know their vote has been counted
		voteData.Voters[userID] = true
		err := c.Discord.SendDM(userID, "Thank you! Your vote has been recorded.")
		if err != nil {
			c.Logger.Errorf("failed to send DM: %s", err.Error())
		}
		if len(voteData.Voters) < c.Config.Voting.RequiredVotes.PanicBan {
			return
		}
		// Delete the vote tracking
		bannedUser, err := c.Discord.GetGuildMemberUsername(voteData.TargetUser)
		if err != nil {
			c.Logger.Errorf("could not find guild member's username %s", err.Error())
		}
		err = c.Discord.BanUser(voteData.TargetUser, voteData.BanReason, int(voteData.Days))
		if err != nil {
			c.Logger.Errorf("failed to ban user: %s", err.Error())
			return
		}
		err = c.Alert("")
		if err != nil {
			c.Logger.Errorf("failed to alert the authorities: %s", err.Error())
		}
		err = c.Discord.SendChannelMessage("", fmt.Sprintf("User %s has been banned. Crisis averted.", bannedUser))
		if err != nil {
			c.Logger.Errorf("failed to notify channel of vote result: %s", err.Error())
		}
		delete(c.VoteTracker, voteID)
	default:
		c.Logger.Errorf("Unknown panic vote type %s", voteData.PanicType)
	}
}
func (c *Container) Alert(message string) error {
	// TODO Send any DMs that need to be sent out
	// TODO Send any Emails that need to be sent out
	// TODO Send any Texts that need to be sent out
	return nil
}

func hasVotePermissions(userID string, userRoles []string, allowedUserIDs []string, allowedUserRoles []string) bool {
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
	c.VoteTracker = make(map[string]VoteData)
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
		RoleRemovedCallback:   c.RoleRemovedCallback,
	})

	if err != nil {
		c.Logger.Fatalf("failed to create Discord session: %s", err)
	}
	c.Twilio, err = panicbot.NewTwilio(&panicbot.TwilioImplArgs{
		AccountSID:        c.Config.AlertingMethods.Twilio.AccountSID,
		APIKey:            c.Config.AlertingMethods.Twilio.APIKey,
		APISecret:         c.Config.AlertingMethods.Twilio.APISecret,
		TwilioPhoneNumber: c.Config.AlertingMethods.Twilio.TwilioPhoneNumber,
		Logger:            c.Logger,
	})

	if err != nil {
		c.Logger.Fatalf("failed to create Discord session: %s", err)
	}
	c.Twilio, err = panicbot.NewTwilio(&panicbot.TwilioImplArgs{
		AccountSID:        c.Config.AlertingMethods.Twilio.AccountSID,
		APIKey:            c.Config.AlertingMethods.Twilio.APIKey,
		APISecret:         c.Config.AlertingMethods.Twilio.APISecret,
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
