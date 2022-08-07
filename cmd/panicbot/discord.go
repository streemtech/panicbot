package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func (c *Container) registerSlashCommands() error {
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
