package main

import (
	"fmt"
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
		return fmt.Errorf("DiscordBotToken cannot be empty, did you forget to set it in the config?")
	}
	return nil
}
