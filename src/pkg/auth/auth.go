package auth

import (
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"utk-auth-go/src/pkg/authserver"
)

func init() {
	// load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}
}

// Discord command metadata
var (
	// name that the command is invoked by
	Name = "auth"

	// invoked by "/auth [netid]"
	Command = discordgo.ApplicationCommand{
		Name:        "auth",
		Description: "Authenticate with your NetID as a student",

		Type: discordgo.ChatApplicationCommand,
		// single argument for the user's NetID
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "netid",
				Description: "Your NetID",
				Required:    true,
			},
		},
	}
)

// SMTPConfig holds configuration for SMTP server
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Sender   string
}

// AuthService handles the authentication logic
type AuthService struct {
	smtpConfig SMTPConfig
}

// PreAuthUser holds the data for a user before they are authenticated
type PreAuthUser struct {
	DiscordUserId  string
	DiscordGuildId string
	NetId          string
}

func NewPreAuthUser(discordUserId string, discordGuildId, netId string) *PreAuthUser {
	preAuthUser := PreAuthUser{
		DiscordUserId:  discordUserId,
		DiscordGuildId: discordGuildId,
		NetId:          netId,
	}
	PreAuthUsers[discordUserId] = &preAuthUser
	return &preAuthUser
}

func NewAuthService() *AuthService {
	return &AuthService{
		smtpConfig: SMTPConfig{
			Host:     "smtp.gmail.com",
			Port:     587,
			Username: os.Getenv("SMTP_USERNAME"),
			Password: os.Getenv("SMTP_PASSWORD"),
			Sender:   os.Getenv("SMTP_USERNAME"),
		},
	}
}

func RequestAuthUrl(preAuthUser *PreAuthUser) string {
	authServerUrl := os.Getenv("AUTH_SERVER_URL")

	// send request to endpoint /generate-user-token
	log.Println("Generating HTTP request")
	requestString := fmt.Sprintf("/generate-user-token?user-discord-id=%s&guild-discord-id=%s", preAuthUser.DiscordUserId, preAuthUser.DiscordGuildId)
	req, err := http.NewRequest("POST", authServerUrl+requestString, nil)
	if err != nil {
		log.Println(err)
	}

	log.Println("Setting request headers")
	req.Header.Set("X-Custom-Auth", os.Getenv("SHARED_SECRET"))
	req.Header.Set("Content-Type", "application/json")

	log.Println("Sending request to authentication server")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return "[Something went wrong while sending the request to the authentication server. Try again.]"
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var response authserver.ApiResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Println("Response body causing error:", body)
		log.Fatal(err)
	}

	dataMap, ok := response.Data.(map[string]interface{})
	if !ok {
		log.Fatal("Data is not the expected type")
	}

	// Access the token within the map
	token, ok := dataMap["token"].(string)
	if !ok {
		log.Fatal("Token is not a string or not present")
	}

	return fmt.Sprintf("%s/verify?user-discord-id=%s&token=%s", authServerUrl, preAuthUser.DiscordUserId, token)
}

func (service *AuthService) SendAuthEmail(netID string, preAuthUser *PreAuthUser, verificationUrl string) error {
	recipient := fmt.Sprintf("%s@vols.utk.edu", netID)
	subject := "UTK COSC Authentication Email"
	body := fmt.Sprintf("Hello %s,\n\n"+
		"Please click the link below to verify your Discord account with UTK."+
		"\n\n%s"+
		"\n\nThank you,"+
		"\nUTK COSC Discord Bot", netID, verificationUrl)

	return service.sendEmail(recipient, subject, body)
}

func (service *AuthService) sendEmail(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", service.smtpConfig.Host, service.smtpConfig.Port)
	auth := smtp.PlainAuth("", service.smtpConfig.Username, service.smtpConfig.Password, service.smtpConfig.Host)

	message := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", to, subject, body))

	return smtp.SendMail(addr, auth, service.smtpConfig.Sender, []string{to}, message)
}

var PreAuthUsers = make(map[string]*PreAuthUser)
