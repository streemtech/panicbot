package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func (c *Container) registerSlashCommands() error {
	c.Logger.Debugf("registering slash commands")
	var def bool = false
	_, err := c.Discord.ApplicationCommandCreate(c.Discord.State.User.ID, c.Config.GuildID, &discordgo.ApplicationCommand{
		Name:              "panicban",
		Description:       "Initializes a panic ban vote.",
		DefaultPermission: &def,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "The user whom the ban vote is about.",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "reason",
				Description: "Reason why this user should be banned.",
				Required:    true,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create panic ban slash command: %s. Make sure that the bot has joined your server with the correct permissions", err.Error())
	}
	_, err = c.Discord.ApplicationCommandCreate(c.Discord.State.User.ID, c.Config.GuildID, &discordgo.ApplicationCommand{
		Name:              "panicalert",
		Description:       "Initializes an alert admin vote.",
		DefaultPermission: &def,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "message",
				Description: "The message to send to the admin.",
				Required:    true,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create panic alert slash command: %s. Make sure that the bot has joined your server with the correct permissions", err.Error())
	}
	return nil
}

// TODO Add handler to deal with slash commands
func (c *Container) handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Hey there! Congratulations, you just executed your first slash command",
		},
	})
}

func (c *Container) findPrimaryChannelInGuild(s *discordgo.Session) (*discordgo.Channel, error) {
	// The primary channel may be provided to us in the config.yml
	// TODO: Add primaryChannelID to config.yml and uncomment code below.
	// if c.Config.primaryChannelID != "" {
	// 	channel, err := c.Discord.Channel(c.Config.primaryChannelID)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to find channel with provided identifier. Is primaryChannelID a valid Discord channel ID?: %s", err)
	// 	}
	// 	// This will allow our users to decide which channel the bot should send its welcome message in.
	// 	return channel, nil
	// }

	guild, err := s.Guild(c.Config.GuildID)
	if err != nil {
		return nil, fmt.Errorf("failed to find guild with provided identifier. Did you forget to put the GuildID in the config?: %s", err)
	}
	channels, err := c.Discord.GuildChannels(c.Config.GuildID)
	if err != nil {
		return nil, fmt.Errorf("failed to find channels in the guild")
	}

	for _, guildChannel := range channels {
		if guildChannel.ID == guild.ID {
			return guildChannel, nil
		}
	}
	// This should never happen as every Discord server should have
	// a primary channel
	return nil, nil
}

func (c *Container) onBotStartup() error {
	c.Logger.Info("running bot startup")

	c.Logger.Debugf("attaching slash command handler")
	c.Discord.AddHandler(c.handleCommand)

	c.Logger.Debugf("reloading roles from config")
	err := c.reloadRoles()
	if err != nil {
		return fmt.Errorf("failed to reload config roles")
	}

	primaryChannel, err := c.findPrimaryChannelInGuild(c.Discord)
	if err != nil {
		return fmt.Errorf("failed to determine primary channel in guild")
	}
	message, err := c.Discord.ChannelMessageSend(primaryChannel.ID, "Hello! Thank you for inviting me!")
	if err != nil {
		return fmt.Errorf("failed to send welcome message: %s", message.Content)
	}
	c.Logger.Infof("successfully sent welcome message: %s", message.Content)
	return nil
}
