package authserver

import (
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"utk-auth-go/src/pkg/auth"
	"utk-auth-go/src/pkg/canvas"
	"utk-auth-go/src/pkg/common"
	"utk-auth-go/src/pkg/utils"
)

var (
	mutex   sync.Mutex
	session *discordgo.Session
)

type ApiResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func init() {
	// setup dotenv
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}
}

type TokenResponse struct {
	Token string `json:"token"`
}

func startCanvasOAuthHandler(w http.ResponseWriter, r *http.Request) {
	var (
		canvasClientID = common.CanvasClientID
		authServerUrl  = common.AuthServerUrl
	)

	// get guild-id from query params
	guildID := r.URL.Query().Get("guild_id")
	if guildID == "" {
		http.Error(w, "Guild ID not found", http.StatusBadRequest)
	}

	discordUserID := r.URL.Query().Get("discord_user_id")
	if discordUserID == "" {
		http.Error(w, "Discord user ID not found", http.StatusBadRequest)
	}

	// Define the Canvas OAuth URL with necessary parameters
	canvasOAuthURL := getCanvasAuthURL(canvasClientID, authServerUrl+"/canvas-callback?guild_id="+guildID+"&discord_user_id="+discordUserID, "1")

	// Redirect the user to Canvas OAuth URL
	http.Redirect(w, r, canvasOAuthURL, http.StatusFound)
}

func getCanvasAuthURL(clientID, redirectURI, state string) string {
	return fmt.Sprintf(common.CanvasInstallUrl+"/login/oauth2/auth?client_id=%s&response_type=code&redirect_uri=%s&state=%s",
		url.QueryEscape(clientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(state))
}

func canvasCallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code not found", http.StatusBadRequest)
		return
	}

	guildID := r.URL.Query().Get("guild_id")
	if guildID == "" {
		http.Error(w, "Guild ID not found", http.StatusBadRequest)
		return
	}

	discordUserID := r.URL.Query().Get("discord_user_id")
	if discordUserID == "" {
		http.Error(w, "Discord user ID not found", http.StatusBadRequest)
		return
	}

	var (
		canvasClientId     = common.CanvasClientID
		canvasClientSecret = common.CanvasClientSecret
		authServerUrl      = common.AuthServerUrl
	)

	// Exchange the code for a Canvas access token
	canvasAccessToken, err := exchangeCanvasCodeForAccessToken(code, canvasClientId, canvasClientSecret, authServerUrl+"/canvas-callback")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	enrollments := canvas.GetUserEnrollments(canvasAccessToken)
	isEnrolled, err := auth.IsEnrolled(enrollments, guildID)
	if err != nil {
		http.Error(w, "Error checking enrollment", http.StatusInternalServerError)
		return
	}
	if !isEnrolled {
		http.Error(w, "You are not enrolled in this course", http.StatusUnauthorized)
		return
	}

	mutex.Lock()
	authRoleID, err := utils.GetAuthRoleID(guildID)
	if err != nil {
		http.Error(w, "Error getting role ID", http.StatusInternalServerError)
		return
	}
	mutex.Unlock()

	if authRoleID == "" {
		http.Error(w, "No role is registered for this server", http.StatusNotFound)
		return
	}

	// grant auth role to user
	err = session.GuildMemberRoleAdd(guildID, discordUserID, authRoleID)
	if err != nil {
		log.Println("Error adding role to user:", err)
	}

	// get name from user in enrollment data
	name := enrollments[0].User.Name

	// change user nickname
	err = session.GuildMemberNickname(guildID, discordUserID, name)
	if err != nil {
		log.Println("Error changing user nickname:", err)
	}
}

func exchangeCanvasCodeForAccessToken(code, clientID, clientSecret, redirectURI string) (string, error) {
	req, err := http.NewRequest("POST", "https://canvas.instructure.com/api/v1/login/oauth2/token", nil)
	if err != nil {
		return "", err
	}

	q := req.URL.Query()
	q.Add("grant_type", "authorization_code")
	q.Add("client_id", clientID)
	q.Add("client_secret", clientSecret)
	q.Add("redirect_uri", redirectURI)
	q.Add("code", code)
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var data map[string]interface{}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", err
	}

	return data["access_token"].(string), nil
}

func StartServer(sessionPass *discordgo.Session) {
	session = sessionPass
	port := os.Getenv("PORT")
	http.HandleFunc("/canvas-login", startCanvasOAuthHandler)
	http.HandleFunc("/canvas-callback", canvasCallbackHandler)

	fmt.Println("Server is running on port", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
