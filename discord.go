package panicbot

import (
	"fmt"
	"time"

	"github.com/k0kubun/pp/v3"
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
	if channelID == "" {
		channelID = Discord.primaryChannelID
	}
	message, err := Discord.session.ChannelMessageSend(channelID, content)
	if err != nil {
		return nil, fmt.Errorf("failed to send message to channel")
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
	panicAlertCallback    func(message string)
	panicBanCallback      func()
}

type DiscordImplArgs struct {
	BotToken              string
	GuildID               string
	PrimaryChannelID      string
	Logger                *log.Logger
	Session               *discordgo.Session
	EmbedReactionCallback func()
	PanicAlertCallback    func(message string)
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
func (Discord *DiscordImpl) handleInteractions(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Step 1: Figure out which one of the three interactions just happened.
	switch i.Interaction.Type {
	case 2:
		if i.ApplicationCommandData().Name == "panicalert" {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					// Here is where we would create the JSON Payload for an embedded message.
					// A listener for the InteractionMessageComponent has already been added.
					// So theoretically, whenever a button is clicked on we can respond to it with the embedButtonCallback.
					Content: "Beginning panic alert vote",
				},
			})
			Discord.panicAlertCallback("A panic alert has started")
		}
		if i.ApplicationCommandData().Name == "panicban" {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Beginning panic ban vote",
				},
			})
			Discord.panicBanCallback()
		}
	// This makes the assumption that an InteractionMessageComponent event is fired whenever an embedded button is clicked on.
	// Because a button is a component of a message.
	case 3:
		Discord.embedReactionCallback()
	}
	// Step 2: Pull the data from the interaction that we care about(going to depend on which interaction)
	// Step 3: Pass that information to the matching callback.
	// Step 4: ? Handle the response to the command so that discord doesn't error. We should not pass the session or the interaction create to the callbacks.
	pp.Println(i)
}

func (Discord *DiscordImpl) registerSlashCommands() error {
	Discord.logger.Infof("registering slash commands")
	var def bool = false
	// Create an array of pointers to discordgo.ApplicationCommand structs
	commands := []*discordgo.ApplicationCommand{
		{
			Name:              "panicalert",
			Description:       "Start an alert admin vote.",
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
		{
			Name:              "panicban",
			Description:       "The user whom the ban vote is about.",
			DefaultPermission: &def,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "Name of the user to ban",
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
		},
	}

	// Add a listener for when the Discord API fires an InteractionCreate event.
	Discord.session.AddHandler(Discord.handleInteractions)
	for _, v := range commands {
		_, err := Discord.session.ApplicationCommandCreate(Discord.session.State.User.ID, Discord.guildID, v)
		if err != nil {
			return fmt.Errorf("cannot create '%v' command: %v", v.Name, err)
		}
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
