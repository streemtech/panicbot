package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// func (c *Container) reloadConfig(newConfig Config) (err error) {
// 	oldConfig := c.Config
// 	if oldConfig.DiscordBotToken != newConfig.DiscordBotToken {
// 		c.Logger.Errorf("DiscordBotToken changed. User must restart this program.")
// 	}
// 	// TODO Check for role changing
// 	err = c.reloadRoles()
// 	if err != nil {
// 		c.Logger.Fatalf("failed to load users in roles: %s", err.Error())
// 	}

// 	return nil
// }

func (c *Container) loadConfig(newConfig Config) (err error) {
	c.Logger.Debugf("begin loading config")
	c.Config = newConfig

	if c.Config.DiscordBotToken == "" {
		return fmt.Errorf("DiscordBotToken field cannot be empty")
	}
	c.Logger.Debugf("preparing session to Discord")
	c.Discord, err = discordgo.New("Bot " + c.Config.DiscordBotToken)
	if err != nil {
		return fmt.Errorf("failed to prepare session to Discord")
	}

	c.Logger.Debugf("opening websocket connection to Discord")
	err = c.Discord.Open()
	if err != nil {
		return fmt.Errorf("failed to open websocket connection to Discord")
	}
	c.Logger.Infof("successfully opened websocket connection to Discord")
	c.onBotStartup()

	return nil
}
