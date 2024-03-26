package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"utk-auth-go/src/pkg/auth"
	"utk-auth-go/src/pkg/authserver"
	"utk-auth-go/src/pkg/utils"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
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
		&auth.Command,
		&utils.RegisterCourseCommand,
	}

	// define command handlers
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		// defer interaction timeout
		auth.Name: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
			if exists, err := utils.GuildIdExists(i.GuildID); err != nil {
				return
			} else if !exists {
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

			netid := i.ApplicationCommandData().Options[0].StringValue()

			// check if student exists in canvas course
			if exists, err := utils.StudentExists(i.GuildID, netid); err != nil {
				return
			} else if !exists {
				log.Println(netid, "is not enrolled in the course.")
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: utils.StrPtr(""),
					Embeds: utils.NewEmbeds(
						utils.NewEmbed(
							"Authentication",
							"You are not enrolled in the course.",
							0xff4400,
							nil,
						),
					),
				})
				return
			}

			// check if student is already authenticated
			if course, err := utils.GetCourseObject(i.GuildID); err != nil {
				log.Println("Something went wrong while getting course object for guildId:", i.GuildID)
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: utils.StrPtr(""),
					Embeds: utils.NewEmbeds(
						utils.NewEmbed(
							"Authentication",
							"Something went wrong while checking your verification status.",
							0xff4400,
							nil,
						),
					),
				})
				return
			} else {
				authRoleID := course.AuthRoleId
				log.Println(i.User.Username + " role list: ")
				for _, role := range i.Member.Roles {
					log.Printf("   Role: %s\n", role)
				}
				for _, role := range i.Member.Roles {
					if role == authRoleID {
						log.Println("User is already authenticated")
						s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
							Content: utils.StrPtr(""),
							Embeds: utils.NewEmbeds(
								utils.NewEmbed(
									"Authentication",
									"You've already been verified.",
									0xff4400,
									nil,
								),
							),
						})
						return
					}
				}
			}

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

				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: utils.StrPtr(""),
					Embeds: utils.NewEmbeds(
						utils.NewEmbed(
							"Authentication",
							"Something went wrong while sending the aunthentication email to NetID: "+netid,
							0xff4400,
							nil,
						),
					),
				})
			}

			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: utils.StrPtr(""),
				Embeds: utils.NewEmbeds(
					utils.NewEmbed(
						"Authentication",
						"An email has been sent to your NetID with a link to authenticate.",
						0xff4400,
						[]*discordgo.MessageEmbedField{
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
					),
				),
			})
		},

		// register course command
		utils.RegisterCourseName: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
				guildId      = i.GuildID
				canvasSecret = i.ApplicationCommandData().Options[0].StringValue()
				courseId     = i.ApplicationCommandData().Options[1].StringValue()
				authRoleId   = i.ApplicationCommandData().Options[2].StringValue()
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

			if exists, err := utils.GuildIdExists(guildId); err != nil {
				return
			} else if exists {
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: &existsString,
				})
				log.Println("Course already registered for this server")
				return
			}

			{
				err := utils.RegisterCourse(guildId, canvasSecret, courseId, authRoleId)
				if err != nil {
					s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
						Content: &failString,
					})
					log.Println(err)
					return
				}
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
