package main

import (
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"github.com/twilio/twilio-go"
	"gopkg.in/yaml.v3"
)

type Boot interface {
	configChanged() error
	configureLogger()
	watchFile(filePath string) error
}
type Config struct {
	Discord_Bot_Token string
}

type Container struct {
	Config           *Config
	Logger           *log.Logger
	Session          *discordgo.Session
	TwilioRestClient *twilio.RestClient
}

var _ Boot = (*Container)(nil)

func main() {
	c := new(Container)
	c.configureLogger()
	err := c.configChanged()
	if err != nil {
		c.Logger.Fatalf("failed to load config: %s", err.Error())
	}
	err = c.watchFile("./config.yml")
	if err != nil {
		c.Logger.Fatalf("failed to watch configuration file: %s", err.Error())
	}
}
func (c *Container) configChanged() error {
	yfile, err := os.ReadFile("./config.yml")
	if err != nil {
		c.Logger.Fatalf("failed to read config file: %s", err.Error())
	}

	c.Config = new(Config)

	err = yaml.Unmarshal(yfile, &c.Config)
	if err != nil {
		c.Logger.Fatalf("failed to unmarshal config data: %s", err.Error())
	}

	return nil
}
func (c *Container) watchFile(filePath string) error {
	// Code taken from: https://levelup.gitconnected.com/how-to-watch-for-file-change-in-golang-4d1eaa3d2964
	// Create a new file watcher.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create config filewatcher: %w", err)
	}
	defer watcher.Close()
	_, err = os.Stat(filePath)
	if os.IsNotExist(err) {
		file, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create file at filePath (%s) for filewatcher: %w", filePath, err)
		}
		file.Close()
	} else if err != nil {
		return fmt.Errorf("failed to stat file at filePath (%s) for filewatcher: %w", filePath, err)
	}
	err = watcher.Add(filePath)
	if err != nil {
		return fmt.Errorf("failed to add filePath (%s) to filewatcher: %w", filePath, err)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return fmt.Errorf("filewatcher events channel closed")
			}
			log.WithFields(log.Fields{
				"Name":      event.Name,
				"Operation": event.Op.String(),
			}).Debug("File event occurred")
			if event.Op == fsnotify.Write {
				err = c.configChanged()
				if err != nil {
					return fmt.Errorf("failed to update config: %w", err)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return fmt.Errorf("filewatcher errors channel closed")
			}
			return fmt.Errorf("filewatcher error encountered: %w", err)
		}
	}
}

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
