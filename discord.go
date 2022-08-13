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

func PanicAlertCallback(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// TODO write logic for starting a panicalert vote
	// TODO if enough votes then call SendDM method passing the information from the config.ContactOnVote {Discord {}} struct
	// TODO if enough votes then call Twilio API to text/call the number from the config.ContactOnVote {Twilio {}} struct
	// TODO if enough votes then call Email handler to email the addresses from the config.ContactOnVote {Email {}} struct
	// TODO write logic for if vote fails. No one is contacted but perhaps a message is sent to the PrimaryChannel. Use SendChannelMessage
}

func PanicBanCallback(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// TODO write logic for starting a panicban vote
	// TODO if enough votes then call SendDM method passing the information from the config.ContactOnVote {Discord {}} struct
	// TODO if enough votes then call Twilio API to text/call the number from the config.ContactOnVote {Twilio {}} struct
	// TODO if enough votes then call Email handler to email the addresses from the config.ContactOnVote {Email {}} struct
	// TODO if enough votes then call BanUser method
	// TODO write logic for if vote fails. No one is contacted but perhaps a message is sent to the PrimaryChannel. Use SendChannelMessage
}

func EmbedReactionCallback(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// TODO use this for whenever we recieve a reaction to a panicalert / panicban
	// This function will be used to tally up the votes and then take action.
}

type DiscordImpl struct {
	botToken              string
	guildID               string
	primaryChannelID      string
	logger                *log.Logger
	session               *discordgo.Session
	embedReactionCallback func(s *discordgo.Session, i *discordgo.InteractionCreate)
	panicAlertCallback    func(s *discordgo.Session, i *discordgo.InteractionCreate)
	panicBanCallback      func(s *discordgo.Session, i *discordgo.InteractionCreate)
}

type DiscordImplArgs struct {
	BotToken              string
	GuildID               string
	PrimaryChannelID      string
	Logger                *log.Logger
	Session               *discordgo.Session
	EmbedReactionCallback func(s *discordgo.Session, i *discordgo.InteractionCreate)
	PanicAlertCallback    func(s *discordgo.Session, i *discordgo.InteractionCreate)
	PanicBanCallback      func(s *discordgo.Session, i *discordgo.InteractionCreate)
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
		return nil, fmt.Errorf("logger was not initialized: %+v", args.Logger)
	}

	// Initialize the bot, register the slash commands
	session, err := discordgo.New("Bot " + args.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare session to Discord")
	}
	// Create a DiscordImpl with args
	discordImpl := &DiscordImpl{
		botToken:              args.BotToken,
		guildID:               args.GuildID,
		primaryChannelID:      args.PrimaryChannelID,
		logger:                args.Logger,
		embedReactionCallback: args.EmbedReactionCallback,
		panicAlertCallback:    args.PanicAlertCallback,
		panicBanCallback:      args.PanicBanCallback,
		session:               session,
	}

	if discordImpl.primaryChannelID == "" {
		primaryChannel, err := discordImpl.findPrimaryChannelInGuild()
		if err != nil {
			return nil, fmt.Errorf("failed to determine primary channel in guild: %s", err)
		}
		discordImpl.primaryChannelID = primaryChannel
	}

	discordImpl.logger.Info("running bot startup")

	discordImpl.logger.Info("preparing Discord session")

	discordImpl.session, err = discordgo.New("Bot " + discordImpl.botToken)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare Discord session: %s", err)
	}
	discordImpl.logger.Info("opening websocket connection to Discord")

	discordImpl.session.Open()

	if err != nil {
		return nil, fmt.Errorf("failed to open websocket connection to Discord: %s", err)
	}

	discordImpl.logger.Infof("successfully opened websocket connection to Discord")

	discordImpl.logger.Infof("attaching slash command handler")

	// c.Logger.Infof("reloading roles from config")
	// err := c.reloadRoles()
	// if err != nil {
	// 	return fmt.Errorf("failed to reload config roles")
	// }

	err = discordImpl.registerSlashCommands()
	if err != nil {
		return nil, fmt.Errorf("failed to register slash commands: %s", err)
	}

	message, err := discordImpl.SendChannelMessage(discordImpl.primaryChannelID, "Hello! Thank you for inviting me!")
	if err != nil {
		return nil, fmt.Errorf("failed to send welcome message: %s", message.Content)
	}
	discordImpl.logger.Infof("successfully sent welcome message: %s", message.Content)

	return discordImpl, nil
}

func (Discord *DiscordImpl) registerSlashCommands() error {
	Discord.logger.Infof("registering slash commands")
	var def bool = false
	// Create an array of pointers to discordgo.ApplicationCommand structs
	commands := []*discordgo.ApplicationCommand{
		{
			Name:              "user",
			Description:       "The user whom the ban vote is about.",
			DefaultPermission: &def,
			Options: []*discordgo.ApplicationCommandOption{
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
		},
		{
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
		},
	}
	// map the names of the commands to their callback
	commandHandlers := map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"panicalert":    Discord.panicAlertCallback,
		"panicban":      Discord.panicBanCallback,
		"embedReaction": Discord.embedReactionCallback,
	}

	// Add a listener for when the Discord API fires an InteractionCreate event.
	Discord.session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if handler, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			handler(s, i)
		}
	})
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := Discord.session.ApplicationCommandCreate(Discord.session.State.User.ID, Discord.guildID, v)
		if err != nil {
			return fmt.Errorf("cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}
	return nil
}

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
