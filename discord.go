package panicbot

import (
	"fmt"
	"time"

	"github.com/k0kubun/pp/v3"
	log "github.com/sirupsen/logrus"
	"github.com/streemtech/panicbot/internal/logic"
	"github.com/streemtech/panicbot/internal/slice"

	"github.com/bwmarrin/discordgo"
)

type Discord interface {
	BanUser(userID string, reason string, days int) error
	SendChannelMessage(channelID string, message string) error
	SendDMEmbed(userID, content, description, titleText, buttonLabel, buttonID string) error
	SendDM(userID string, message string) error
	GetAllGuildMembers() ([]UserRoles, error)
	GetGuildMemberUsername(userID string) (string, error)
}

type UserRoles struct {
	UserID string
	Roles  []string
}
type AllowedToVote struct {
	PanicAlert struct {
		Users []string
		Roles []string
	}
	PanicBan struct {
		Users []string
		Roles []string
	}
}
type DiscordImpl struct {
	allowedToVote         AllowedToVote
	botToken              string
	guildID               string
	primaryChannelID      string
	logger                *log.Logger
	session               *discordgo.Session
	embedReactionCallback func(userID, buttonID string)
	panicAlertCallback    func(message string)
	panicBanCallback      func(userID, targetUserID, reason string, days float64)
	roleRemovedCallback   func(user, role string)
}

type DiscordImplArgs struct {
	AllowedToVote         AllowedToVote
	BotToken              string
	GuildID               string
	PrimaryChannelID      string
	Logger                *log.Logger
	Session               *discordgo.Session
	EmbedReactionCallback func(userID, buttonID string)
	PanicAlertCallback    func(message string)
	PanicBanCallback      func(userID, targetUserID, reason string, days float64)
	RoleRemovedCallback   func(user, role string)
}

var _ Discord = (*DiscordImpl)(nil)

func (d *DiscordImpl) BanUser(userID string, reason string, days int) error {

	err := d.session.GuildBanCreateWithReason(d.guildID, userID, reason, days)
	if err != nil {
		return fmt.Errorf("failed to ban user with userID:  %s", userID)
	}

	guildBan, err := d.session.GuildBan(d.guildID, userID)
	if err != nil {
		return fmt.Errorf("failed to retrieve ban information for user with userID: %s", userID)
	}

	d.logger.WithFields(log.Fields{
		"user":     guildBan.User.String(),
		"reason":   reason,
		"dateTime": time.Now().String(),
	})

	return nil
}

func (d *DiscordImpl) SendChannelMessage(channelID string, content string) error {
	if channelID == "" {
		channelID = d.primaryChannelID
	}
	message, err := d.session.ChannelMessageSend(channelID, content)
	if err != nil {
		return fmt.Errorf("failed to send message to channel")
	}
	d.logger.WithFields(log.Fields{
		"author":    message.Author,
		"channelID": message.ChannelID,
		"guildID":   message.GuildID,
		"message":   message.Content,
		"messageID": message.ID,
		"dateTime":  time.Now().String(),
	})
	return nil
}

func (d *DiscordImpl) SendDM(userID string, message string) error {
	err := d.SendChannelMessage(userID, message)
	if err != nil {
		return fmt.Errorf("failed to send direct message to user with ID: %s", userID)
	}
	return nil
}

func (d *DiscordImpl) SendDMEmbed(userID, content, description, titleText, buttonLabel, buttonID string) error {
	channel, err := d.session.UserChannelCreate(userID)
	if err != nil {
		return fmt.Errorf("failed to create private message channel with userID: %s", userID)
	}
	message := &discordgo.MessageSend{
		Content: content,
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.Button{
				Label:    buttonLabel,
				Style:    discordgo.DangerButton,
				CustomID: buttonID,
				Emoji:    discordgo.ComponentEmoji{Name: "🔨"},
			}}},
		},

		Embeds: []*discordgo.MessageEmbed{
			{
				Type:        discordgo.EmbedTypeRich,
				Title:       titleText,
				Color:       0xDE3163,
				Description: description,
			},
		},
	}
	_, err = d.session.ChannelMessageSendComplex(channel.ID, message)
	if err != nil {
		return fmt.Errorf("failed to send private message with embed to user with ID: %s: %w", userID, err)
	}
	d.logger.WithFields(log.Fields{
		"channelID": userID,
	}).Info("Sent DM")
	return nil
}

func (d *DiscordImpl) GetAllGuildMembers() ([]UserRoles, error) {
	temp := make([]*discordgo.Member, 0)
	userRoles := make([]UserRoles, 0)
	latestMember := ""
	for {
		// Make a call to GuildMembers
		gm, err := d.session.GuildMembers(d.guildID, latestMember, 1000)
		if err != nil {
			return nil, fmt.Errorf("failed to get guild members from guild with ID: %s", d.guildID)
		}
		// Append the result of the call to GuildMembers to out
		temp = append(temp, gm...)
		// Check to see if the call to guild members is less than 1000
		if len(gm) < 1000 {
			break
		}
		latestMember = gm[999].User.ID
	}
	for _, v := range temp {
		userRoles = append(userRoles, UserRoles{UserID: v.User.ID, Roles: v.Roles})
	}
	return userRoles, nil

}

func (d *DiscordImpl) GetGuildMemberUsername(userID string) (string, error) {
	if userID == "" {
		return "", fmt.Errorf("userID cannot be empty: %s", userID)
	}
	member, err := d.session.GuildMember(d.guildID, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get member username with ID: %s in guild with ID: %s", userID, d.guildID)
	}
	return fmt.Sprintf("%s#%s", member.User.Username, member.User.Discriminator), nil
}

func handlePermissionsBadRequest(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// TODO 3: Track if the user without permissions is doing this multiple times and stop the bot from responding.
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "I'm sorry, you do not have permission to use this command.",
		},
	})
}

func hasCommandPermissions(userIDsAllowedToVote []string, userID string, userRolesAllowedToVote []string, userRoles []string) bool {

	userHasPermissions := slice.Contains(userIDsAllowedToVote, userID)
	roleHasPermissions := false
	// Check each role a user has against the list of userRolesAllowedToVote
	for _, v := range userRoles {
		roleHasPermissions = slice.Contains(userRolesAllowedToVote, v)
	}
	return logic.Or(userHasPermissions, roleHasPermissions)
}

func NewDiscord(args *DiscordImplArgs) (*DiscordImpl, error) {
	// Validate that Guild ID is set. If primaryChannelID is not set calculate it.
	if args.GuildID == "" {
		return nil, fmt.Errorf("GuildID cannot be empty. Did you forget to set it in the config? %s", args.GuildID)
	}

	// Verify the logger is set
	if args.Logger == nil {
		return nil, fmt.Errorf("logger was not initialized: %+v", args.Logger)
	}

	// Validate that all callbacks are set
	if args.EmbedReactionCallback == nil {
		return nil, fmt.Errorf("failed to start bot, EmbedReactionCallback was not passed in")
	}
	if args.PanicAlertCallback == nil {
		return nil, fmt.Errorf("failed to start bot, PanicAlertCallback was not passed in")
	}
	if args.PanicBanCallback == nil {
		return nil, fmt.Errorf("failed to start bot, PanicBanCallback was not passed in")
	}

	args.Logger.Info("preparing Discord session")
	// Initialize the bot, register the slash commands
	session, err := discordgo.New("Bot " + args.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare session to Discord")
	}

	session.StateEnabled = true
	// Create a DiscordImpl with args
	discordImpl := &DiscordImpl{
		allowedToVote:         args.AllowedToVote,
		botToken:              args.BotToken,
		guildID:               args.GuildID,
		primaryChannelID:      args.PrimaryChannelID,
		logger:                args.Logger,
		embedReactionCallback: args.EmbedReactionCallback,
		panicAlertCallback:    args.PanicAlertCallback,
		panicBanCallback:      args.PanicBanCallback,
		roleRemovedCallback:   args.RoleRemovedCallback,
		session:               session,
	}

	if discordImpl.primaryChannelID == "" {
		primaryChannel, err := discordImpl.findPrimaryChannelInGuild()
		if err != nil {
			return nil, fmt.Errorf("failed to determine primary channel in guild: %w", err)
		}
		discordImpl.primaryChannelID = primaryChannel
	}

	discordImpl.logger.Info("running bot startup")

	discordImpl.logger.Info("opening websocket connection to Discord")

	discordImpl.session.Open()

	if err != nil {
		return nil, fmt.Errorf("failed to open websocket connection to Discord: %w", err)
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
		return nil, fmt.Errorf("failed to register slash commands: %w", err)
	}

	err = discordImpl.SendChannelMessage(discordImpl.primaryChannelID, "Hello! Thank you for inviting me!")
	if err != nil {
		return nil, fmt.Errorf("failed to send welcome message: %w", err)
	}
	discordImpl.logger.Infof("successfully sent welcome message")

	return discordImpl, nil
}

func (d *DiscordImpl) handleMemberUpdate(s *discordgo.Session, i *discordgo.GuildMemberUpdate) {
	cachedUser, err := s.State.Member(i.GuildID, i.User.ID)
	if err != nil {
		return
	}
	if cachedUser == nil {
		return
	}
	newRoles := i.Roles
	oldRoles := cachedUser.Roles

	newRoleMap := make(map[string]struct{})
	for _, role := range newRoles {
		newRoleMap[role] = struct{}{}
	}

	removed := make([]string, 0)
	for _, role := range oldRoles {
		//if role is in old roles, but not in new role map, that role was removed.
		//log that and append it to the user
		if _, ok := newRoleMap[role]; !ok {
			removed = append(removed, role)
		}
	}

	//call the callback for each removed role.
	for _, role := range removed {
		d.roleRemovedCallback(cachedUser.User.ID, role)
	}

}

func (d *DiscordImpl) handleInteractions(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Step 1: Figure out which one of the three interactions just happened.
	switch i.Interaction.Type {
	case discordgo.InteractionApplicationCommand:
		if i.ApplicationCommandData().Name == "panicalert" {
			if !hasCommandPermissions(d.allowedToVote.PanicAlert.Users, i.Member.User.ID, d.allowedToVote.PanicAlert.Roles, i.Member.Roles) {
				handlePermissionsBadRequest(s, i)
			} else {
				err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						// Here is where we would create the JSON Payload for an embedded message.
						// A listener for the InteractionMessageComponent has already been added.
						// So theoretically, whenever a button is clicked on we can respond to it with the embedButtonCallback.
						Content: "Beginning panic alert vote",
					},
				})
				if err != nil {
					d.logger.Errorf("failed to respond to application command: %w", err)
					return
				}
				d.panicAlertCallback(i.ApplicationCommandData().Options[0].Value.(string))
			}
		}
		if i.ApplicationCommandData().Name == "panicban" {
			slashCommandData := i.ApplicationCommandData()
			if !hasCommandPermissions(d.allowedToVote.PanicBan.Users, i.Member.User.ID, d.allowedToVote.PanicBan.Roles, i.Member.Roles) {
				handlePermissionsBadRequest(s, i)
			} else {
				err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "🚨A Panic Ban vote has started! Voters check your DMs. This message will self-destruct in one second.🚨",
					},
				})
				if err != nil {
					d.logger.Errorf("failed to respond to application command: %w", err)
					return
				}
				time.AfterFunc(time.Second*1, func() {
					s.InteractionResponseDelete(i.Interaction)
				})
				d.panicBanCallback(i.Interaction.Member.User.ID, slashCommandData.Options[0].Value.(string), slashCommandData.Options[1].Value.(string), slashCommandData.Options[2].Value.(float64))
			}
		}
	// This makes the assumption that an InteractionMessageComponent event is fired whenever an embedded button is clicked on.
	// Because a button is a component of a message.
	case discordgo.InteractionMessageComponent:
		dat, ok := i.Interaction.Data.(discordgo.MessageComponentInteractionData)
		if !ok {
			d.logger.Errorf("Interaction Data of unexpected type, %T", i.Interaction.Data)
			break
		}
		if i.Interaction.User == nil {
			d.logger.Errorf("Unable to get user from interaction")
			break
		}
		d.embedReactionCallback(i.Interaction.User.ID, dat.CustomID)
	}
	// Step 2: Pull the data from the interaction that we care about(going to depend on which interaction)
	// Step 3: Pass that information to the matching callback.
	// Step 4: ? Handle the response to the command so that discord doesn't error. We should not pass the session or the interaction create to the callbacks.
	pp.Println(i)
}

func (d *DiscordImpl) registerSlashCommands() error {
	d.logger.Infof("registering slash commands")
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
	d.session.AddHandler(d.handleInteractions)
	d.session.AddHandler(d.handleMemberUpdate)
	for _, v := range commands {
		_, err := d.session.ApplicationCommandCreate(d.session.State.User.ID, d.guildID, v)
		if err != nil {
			return fmt.Errorf("cannot create '%v' command: %v", v.Name, err)
		}
	}
	return nil
}

func (d *DiscordImpl) findPrimaryChannelInGuild() (string, error) {
	// The primary channel may be provided to us in the config.yml
	if d.primaryChannelID != "" {
		channel, err := d.session.Channel(d.primaryChannelID)
		if err != nil {
			return "", fmt.Errorf("failed to find channel with provided identifier. Is primaryChannelID a valid Discord channelID?: %w", err)
		}
		return channel.ID, nil
	}

	guild, err := d.session.Guild(d.guildID)
	if err != nil {
		return "", fmt.Errorf("failed to find guild with provided identifier. Did you forget to put the GuildID in the config?: %w", err)
	}

	channels, err := d.session.GuildChannels(d.guildID)
	if err != nil {
		return "", fmt.Errorf("failed to find channels in the guild: %w", err)
	}

	for _, guildChannel := range channels {
		if guildChannel.ID == guild.ID {
			return guildChannel.ID, nil
		}
	}

	return "", nil
}
