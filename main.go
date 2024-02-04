package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"log"
	"os"
	"utk-auth-go/src/pkg/auth"
	"utk-auth-go/src/pkg/authserver"
	"utk-auth-go/src/pkg/registercourse"
	"utk-auth-go/src/pkg/utils"
)

var session *discordgo.Session

func init() {
	{
		err := godotenv.Load()
		if err != nil {
			log.Println("Error loading .env file")
		}
	}

	{
		// create file server_config.json if it doesn't exist
		if _, err := os.Stat("/data/server_config.json"); os.IsNotExist(err) {
			file, err := os.Create("/data/server_config.json")
			if err != nil {
				log.Println("Error creating server_config.json")
			}
			defer file.Close()
		}
	}
}

var (
	DiscordToken = os.Getenv("DISCORD_TOKEN")
)

// initialize bot
func init() {
	var err error
	session, err = discordgo.New("Bot " + os.Getenv("DISCORD_TOKEN"))
	if err != nil {
		log.Fatal("Error creating Discord session")
	}
	session.Identify.Intents = discordgo.IntentsAllWithoutPrivileged | discordgo.IntentsMessageContent | discordgo.IntentsGuildMembers
}

// initialize bot commands
var (
	commands = []*discordgo.ApplicationCommand{
		&auth.AuthCommand,
		&registercourse.RegisterCourseCommand,
	}

	// define command handlers
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		// defer interaction timeout
		"auth": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{
						{
							Title:       "Authentication",
							Description: "Attempting to authenticate...",
							Color:       0xff4400,
						},
					},
					Flags: discordgo.MessageFlagsEphemeral,
				},
			})

			// check if guild exists before initiating authentication
			if exists := utils.GuildIDExists(i.GuildID); !exists {
				log.Println("No course is registered for this server")
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: utils.StrPtr(""),
					Embeds: utils.NewEmbeds(
						utils.NewEmbed(
							"Authentication",
							"No course is registered for this server.\nPlease use `/registercourse` to register a course.",
							0xff4400,
							nil,
						),
					),
				})
				return
			}

			authURL := "utk-auth-go-production.up.railway.app/canvas-login?guild_id=" + i.GuildID + "&discord_user_id=" + i.Member.User.ID
			// edit embed with OAuth URL
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: utils.StrPtr(""),
				Embeds: utils.NewEmbeds(
					utils.NewEmbed(
						"Authentication",
						"Click [this link]"+"("+authURL+") to authenticate with your NetID.",
						0xff4400,
						nil,
					),
				),
			})
		},

		// register course command
		"registercourse": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// responding immedately to defer interaction timeout
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Attempting to register course...",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})

			// interaction message strings for editing after server response(s)
			var (
				successString = "Course registered successfully!"
				failString    = "Failed to register course, something went wrong."
				existsString  = "Course already registered for this server."
			)

			var (
				guildId    = i.GuildID
				courseId   = i.ApplicationCommandData().Options[1].StringValue()
				authRoleId = i.ApplicationCommandData().Options[2].StringValue()
			)

			// memberRoles := i.Member.Roles
			// isAdmin := false
			// for _, role := range memberRoles {
			// 	if role == "admin" {
			// 		isAdmin = true
			// 	}
			// }

			// if !isAdmin {
			// 	log.Println("User does not have permission to register a course for this server")
			// 	return
			// }

			if exists := utils.GuildIDExists(guildId); exists {
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: &existsString,
				})
				log.Println("Course already registered for this server")
				return
			}

			err := registercourse.RegisterCourse(guildId, courseId, authRoleId)
			if err != nil {
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: &failString,
				})
				return
			}

			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &successString,
			})
		},
	}
)

// initialize bot handlers
func init() {
	session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}

func main() {
	// Setup HTTP server
	var err error
	err = session.Open()
	if err != nil {
		log.Fatal(err)
	}

	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := session.ApplicationCommandCreate(session.State.User.ID, "", v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}
	defer session.Close()

	go func() {
		authserver.StartServer(session)
	}()
	fmt.Println("Bot is now running. Press CTRL+C to exit.")

	// Wait here until CTRL+C or other term signal is received.
	<-make(chan struct{})
	return
}
