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
	c.Config = newConfig
	if c.Config.DiscordBotToken == "" {
		return fmt.Errorf("DiscordBotToken field cannot be empty")
	}
	c.Discord, err = discordgo.New("Bot " + c.Config.DiscordBotToken)
	if err != nil {
		return fmt.Errorf("failed to prepare connection to Discord")
	}
	c.Discord.AddHandler(c.handleCommand)
	err = c.Discord.Open()
	if err != nil {
		return fmt.Errorf("failed to open connection to Discord")
	}
	err = c.reloadRoles()
	if err != nil {
		return fmt.Errorf("failed to reload roles")
	}

	return nil
}
