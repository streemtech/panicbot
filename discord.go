package panicbot

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bwmarrin/discordgo"
)

type Discord interface {
	BanUser(userID string, reason string, days int) (discordgo.GuildBan, error)
	SendChannelMessage(channelID string, message string) (*discordgo.Message, error)
	SendDM(userID string, message string) (*discordgo.Message, error)
}

func (Discord *DiscordImpl) BanUser(userID string, reason string, days int) (discordgo.GuildBan, error) {

	err := Discord.session.GuildBanCreateWithReason(Discord.guildID, userID, reason, days)
	if err != nil {
		return discordgo.GuildBan{}, fmt.Errorf("failed to ban user with userID:  %s", userID)
	}

	guildBan, err := Discord.session.GuildBan(Discord.guildID, userID)
	if err != nil {
		return discordgo.GuildBan{}, fmt.Errorf("failed to retrieve ban information for user with userID: %s", userID)
	}

	Discord.logger.WithFields(log.Fields{
		"user":     guildBan.User.String(),
		"reason":   reason,
		"dateTime": time.Now().String(),
	})

	return *guildBan, nil
}

func (Discord *DiscordImpl) SendChannelMessage(channelID string, content string) (*discordgo.Message, error) {
	message, err := Discord.session.ChannelMessageSend(channelID, content)
	if err != nil {
		return nil, fmt.Errorf("failed to send message to primary channel")
	}
	Discord.logger.WithFields(log.Fields{
		"author":    message.Author,
		"channelID": message.ChannelID,
		"guildID":   message.GuildID,
		"message":   message.Content,
		"dateTime":  time.Now().String(),
	})
	return message, nil
}
func (Discord *DiscordImpl) SendDM(userID string, message string) (*discordgo.Message, error) {
	channel, err := Discord.session.UserChannelCreate(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to create private message channel with userID: %s", userID)
	}
	return Discord.SendChannelMessage(channel.ID, message)
}

type DiscordImpl struct {
	botToken              string
	guildID               string
	primaryChannelID      string
	logger                *log.Logger
	session               *discordgo.Session
	embedReactionCallback func()
	panicAlertCallback    func()
	panicBanCallback      func()
}

type DiscordImplArgs struct {
	BotToken              string
	GuildID               string
	PrimaryChannelID      string
	Logger                *log.Logger
	Session               *discordgo.Session
	EmbedReactionCallback func()
	PanicAlertCallback    func()
	PanicBanCallback      func()
}

var _ Discord = (*DiscordImpl)(nil)

func NewDiscord(args *DiscordImplArgs) (*DiscordImpl, error) {
	// Validate that all callbacks are set
	// Validate that Guild ID is set. If primaryChannelID is not set calculate it.
	if args.GuildID == "" {
		return nil, fmt.Errorf("GuildID cannot be empty. Did you forget to set it in the config? %s", args.GuildID)
	}
	// Verify the logger is set
	if args.Logger == nil {
		return nil, fmt.Errorf("logger was not initialized")
	}

	// Initialize the bot, register the slash commands
	session, err := discordgo.New("Bot " + args.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare session to Discord")
	}
	// Create a DiscordImpl with args
	discordImpl := &DiscordImpl{
		botToken:           args.BotToken,
		guildID:            args.GuildID,
		logger:             args.Logger,
		panicAlertCallback: args.PanicAlertCallback,
		panicBanCallback:   args.PanicBanCallback,
		primaryChannelID:   args.PrimaryChannelID,
		session:            session,
	}

	if discordImpl.primaryChannelID == "" {
		primaryChannel, err := discordImpl.findPrimaryChannelInGuild()
		if err != nil {
			return nil, fmt.Errorf("failed to determine primary channel in guild")
		}
		discordImpl.primaryChannelID = primaryChannel
	}
	// Most of the code in onBotStartup gets moved here.
	discordImpl.logger.Info("running bot startup")

	discordImpl.logger.Debugf("attaching slash command handler")

	// c.Logger.Debugf("reloading roles from config")
	// err := c.reloadRoles()
	// if err != nil {
	// 	return fmt.Errorf("failed to reload config roles")
	// }

	message, err := discordImpl.SendChannelMessage(discordImpl.primaryChannelID, "Hello! Thank you for inviting me!")
	if err != nil {
		return nil, fmt.Errorf("failed to send welcome message: %s", message.Content)
	}
	discordImpl.logger.Infof("successfully sent welcome message: %s", message.Content)

	return discordImpl, nil
}

func (Discord *DiscordImpl) registerSlashCommands() error {
	Discord.logger.Debugf("registering slash commands")
	var def bool = false
	_, err := Discord.session.ApplicationCommandCreate(Discord.session.State.User.ID, Discord.guildID, &discordgo.ApplicationCommand{
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
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "days",
				Description: "The number of days of previous messages to delete",
				Required:    false,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create panic ban slash command: %s. Make sure that the bot has joined your server with the correct permissions", err.Error())
	}
	_, err = Discord.session.ApplicationCommandCreate(Discord.session.State.User.ID, Discord.guildID, &discordgo.ApplicationCommand{
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

// // TODO Add handler to deal with slash commands
// func (Discord *DiscordImpl) handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
// 	// Parse the information from the i
// 	// Call the callback with s and information needed.
// 	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
// 		Type: discordgo.InteractionResponseChannelMessageWithSource,
// 		Data: &discordgo.InteractionResponseData{
// 			Content: "Hey there! Congratulations, you just executed your first slash command",
// 		},
// 	})
// }
func (Discord *DiscordImpl) findPrimaryChannelInGuild() (string, error) {
	// The primary channel may be provided to us in the config.yml
	if Discord.primaryChannelID != "" {
		channel, err := Discord.session.Channel(Discord.primaryChannelID)
		if err != nil {
			return "", fmt.Errorf("failed to find channel with provided identifier. Is primaryChannelID a valid Discord channelID?: %s", err)
		}
		return channel.ID, nil
	}

	guild, err := Discord.session.Guild(Discord.guildID)
	if err != nil {
		return "", fmt.Errorf("failed to find guild with provided identifier. Did you forget to put the GuildID in the config?: %s", err)
	}

	channels, err := Discord.session.GuildChannels(Discord.guildID)
	if err != nil {
		return "", fmt.Errorf("failed to find channels in the guild")
	}

	for _, guildChannel := range channels {
		if guildChannel.ID == guild.ID {
			return guildChannel.ID, nil
		}
	}

	return "", nil
}
