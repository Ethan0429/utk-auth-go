package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"log"
	"os"
	"utk-auth-go/src/pkg/auth"
	"utk-auth-go/src/pkg/authserver"
	"utk-auth-go/src/pkg/utils"
)

var session *discordgo.Session

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
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
		&auth.Command,
		&utils.RegisterCourseCommand,
	}

	// define command handlers
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		auth.Name: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			// check if guild exists before initiating authentication
			if exists, err := utils.GuildIdExists(i.GuildID); err != nil {
				return
			} else if !exists {
        log.Println("No course is registered for this server")
				embed := &discordgo.MessageEmbed{
					Title:       "Authentication",
					Description: "No course is registered for this server.\nPlease use **/registercourse** to register a course.",
					Color:       0xff4400,
				}
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{embed},
						Flags:  discordgo.MessageFlagsEphemeral,
					},
				})
				return
			}
			netid := i.ApplicationCommandData().Options[0].StringValue()

			// send authentication email
			preAuthUser := auth.NewPreAuthUser(i.Member.User.ID, i.GuildID, netid)
			authService := auth.NewAuthService()

			log.Println("Generating authentication URL for NetID:", netid)
			authUrl := auth.RequestAuthUrl(preAuthUser)

			log.Println("Sending authentication email to NetID:", netid)
			err := authService.SendAuthEmail(netid, preAuthUser, authUrl)
			if err != nil {
				log.Println("Something went wrong while sending the aunthentication email to NetID:", netid)
				log.Print(err)
			}

			embed := &discordgo.MessageEmbed{
				Title:       "Authentication",
				Description: "An email has been sent to your NetID with a link to authenticate.",
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Outlook",
						Value:  "**Note**: If you're using Outlook, the email is likely in your **quarantine** folder",
						Inline: false,
					},
					{
						Name:   "Gmail",
						Value:  "**Note**: If you're using Gmail, the email is likely in your **spam** folder",
						Inline: false,
					},
				},
				Color: 0xff4400,
			}

			// respond with ephemeral message
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
					Flags:  discordgo.MessageFlagsEphemeral,
				},
			})
		},
		utils.RegisterCourseName: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			guildId := i.GuildID
			canvasSecret := i.ApplicationCommandData().Options[0].StringValue()
			courseId := i.ApplicationCommandData().Options[1].StringValue()
			authRoleId := i.ApplicationCommandData().Options[2].StringValue()

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

			if exists, err := utils.GuildIdExists(guildId); err != nil {
				return
			} else if exists {
				log.Println("Course already registered for this server")
				return
			}

			{
				err := utils.RegisterCourse(guildId, canvasSecret, courseId, authRoleId)
				if err != nil {
					return
				}
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Course registered successfully!",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
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
