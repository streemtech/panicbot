package main

import (
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
	"github.com/twilio/twilio-go"
	"sigs.k8s.io/yaml"
)

// type Boot interface {
// 	configChanged() error
// 	configureLogger()
// 	watchFile(filePath string) error
// }
type Config struct {
	DiscordBotToken string
	GuildID         string
	RoleID          string
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
}

func (c *Container) configChanged(load bool) error {
	yfile, err := os.ReadFile("./config.yml")
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
		c.Logger.Errorf("Unable to parse log level %s:%s", level, err.Error())
	}
	c.Logger.SetLevel((logLevel))
}
