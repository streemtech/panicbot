package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
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
		return fmt.Errorf("failed to create panic ban slash command: %s", err.Error())
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
		return fmt.Errorf("failed to create panic alert slash command: %s", err.Error())
	}
	return nil
}

// TODO Add handler to deal with reaction voting
func (c *Container) handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Hey there! Congratulations, you just executed your first slash command",
		},
	})
}

func (c *Container) findPrimaryChannelInGuild(s *discordgo.Session, guildID *string) (*discordgo.Channel, error) {
	guild, err := s.Guild(*guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to find primary channel")
	}

	for _, guildChannel := range guild.Channels {
		if guildChannel.ID == guild.ID {
			return guildChannel, nil
		}
	}
	// This should never happen as every Discord server should have
	// a primary channel
	return nil, nil
}

func (c *Container) onBotJoinGuild(s *discordgo.Session, event *discordgo.GuildMemberAdd) {
	c.Logger.WithFields(logrus.Fields{
		"guildID":  event.GuildID,
		"joinedAt": event.JoinedAt,
		"userId":   event.User.ID,
		"username": event.User.Username,
	}).Info("Received guild member add event from Discord Websocket API.")

	primaryChannel, err := c.findPrimaryChannelInGuild(s, &event.GuildID)
	if err != nil {
		c.Logger.WithFields(logrus.Fields{
			"userID":        event.User.ID,
			"guildID":       event.GuildID,
			"capturedError": err,
		}).Error("Could not determine primary channel for guild.")
		return
	}
	s.ChannelMessageSend(primaryChannel.ID, "Hello! Thank you for inviting me!")
}
