package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"log"
	"os"
	"utk-auth-go/src/pkg/auth"
	"utk-auth-go/src/pkg/authserver"
)

var session *discordgo.Session

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
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
	session.Identify.Intents = discordgo.IntentsAllWithoutPrivileged
}

// initialize bot commands
var (
	commands = []*discordgo.ApplicationCommand{
		&auth.Command,
	}

	// define command handlers
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		auth.Name: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
				Color:       0x00ff00,
			}
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Embeds: []*discordgo.MessageEmbed{embed},
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
