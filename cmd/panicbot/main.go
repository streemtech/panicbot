package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/k0kubun/pp/v3"
	log "github.com/sirupsen/logrus"
	"github.com/twilio/twilio-go"
	"os"
	"os/signal"
	"sigs.k8s.io/yaml"
	"time"
)

// TODO: Change debug logs to info.

// type Boot interface {
// 	configChanged() error
// 	configureLogger()
// 	watchFile(filePath string) error
// }
type Config struct {
	DiscordBotToken  string
	GuildID          string
	PrimaryChannelID string
	AlertingMethods  AlertingMethods
	Voting           Voting
}
type AlertingMethods struct {
	Twilio Twilio
	Email  Email
}
type Voting struct {
	ContactOnVote *ContactOnVote
	RequiredVotes struct {
		PanicAlert int
		PanicBan   int
	}
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
	VoteTimers struct {
		PanicAlertVoteTimer time.Duration
		PanicBanVoteTimer   time.Duration
	}
	Cooldown struct {
		PanicAlert time.Duration
		PanicBan   time.Duration
	}
	RateLimit struct {
		PanicAlert struct {
			Day  int
			Hour int
		}
		PanicBan struct {
			Day  int
			Hour int
		}
	}
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
type Twilio struct {
	AccountSID        string
	AuthToken         string
	TwilioPhoneNumber string
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
type Container struct {
	Config           Config
	Logger           *log.Logger
	Discord          *discordgo.Session
	TwilioRestClient *twilio.RestClient
}

// var _ Boot = (*Container)(nil)

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
	err = c.registerSlashCommands()
	if err != nil {
		c.Logger.Fatalf("failed to register slash commands : %s", err.Error())
	}
	// err = c.watchFile("./config.yml")
	// if err != nil {
	// 	c.Logger.Fatalf("failed to watch configuration file: %s", err.Error())
	// }
	defer c.Discord.Close()

	c.Logger.Debugf("create channel to listen for os interrupt")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	c.Logger.Infof("Press Ctrl+C to exit")
	<-stop

	c.Logger.Infof("Gracefully shutting down.")
	pp.Println(c.Config)
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
	c.Logger.SetLevel((logLevel))
	c.Logger.Infof("log level set to: %s", logLevel)
}
