package authserver

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

var mutex sync.Mutex
var session *discordgo.Session

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

// TokenData holds the token and guild ID
type TokenData struct {
	Token   string `json:"token"`
	GuildID string `json:"guild_id"`
}

type TokenResponse struct {
	Token string `json:"token"`
}

// Generate a random token of 25 characters
func generateToken() (string, error) {
	bytes := make([]byte, 25)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[:25], nil
}

// Handler for generating user token
func GenerateUserTokenHandler(w http.ResponseWriter, r *http.Request) {
	sharedSecret := os.Getenv("SHARED_SECRET")

	authHeader := r.Header.Get("X-Custom-Auth")
	if authHeader != sharedSecret {
		json.NewEncoder(w).Encode(ApiResponse{Success: false, Message: "Unauthorized"})
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	userDiscordID := r.URL.Query().Get("user-discord-id")
	guildDiscordID := r.URL.Query().Get("guild-discord-id")

	if userDiscordID == "" || guildDiscordID == "" {
		json.NewEncoder(w).Encode(ApiResponse{Success: false, Message: "Missing parameters"})
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// check if id already exists
	{
		mutex.Lock()
		file, err := os.ReadFile("/data/tokens.json")
		if err != nil {
			if !os.IsNotExist(err) {
				http.Error(w, "tokens.json does not exist", http.StatusInternalServerError)
			}
		} else {
			var tokens map[string]TokenData
			err = json.Unmarshal(file, &tokens)
			if err != nil {
				http.Error(w, "Error parsing file", http.StatusInternalServerError)
			}

			if _, ok := tokens[userDiscordID]; ok {
				http.Error(w, "User already has a token", http.StatusConflict)
				return
			}
		}
		mutex.Unlock()
	}

	token, err := generateToken()
	if err != nil {
		json.NewEncoder(w).Encode(ApiResponse{Success: false, Message: "Error generating token"})
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Read the existing tokens
	mutex.Lock()
	file, err := os.ReadFile("/data/tokens.json")
	if err != nil {
		if !os.IsNotExist(err) {
			http.Error(w, "Error reading file", http.StatusInternalServerError)
			return
		}
		file = []byte("{}") // If the file does not exist, start with an empty JSON object
	}

	var tokens map[string]TokenData
	err = json.Unmarshal(file, &tokens)
	if err != nil {
		http.Error(w, "Error parsing file", http.StatusInternalServerError)
		return
	}
	mutex.Unlock()

	// Add or update the token for the user
	tokens[userDiscordID] = TokenData{Token: token, GuildID: guildDiscordID}

	// Write the updated tokens back to the file
	updatedData, err := json.Marshal(tokens)
	if err != nil {
		http.Error(w, "Error marshalling JSON", http.StatusInternalServerError)
		return
	}

	mutex.Lock()
	err = os.WriteFile("/data/tokens.json", updatedData, 0644)
	if err != nil {
		http.Error(w, "Error writing to file", http.StatusInternalServerError)
		return
	}
	mutex.Unlock()

	response := ApiResponse{Success: true, Message: "Token generated successfully", Data: TokenResponse{Token: token}}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Handler for verifying user token
func VerifyHandler(w http.ResponseWriter, r *http.Request) {
	// Check for GET request to serve the HTML page
	if r.Method == "GET" {
		userDiscordID := r.URL.Query().Get("user-discord-id")
		token := r.URL.Query().Get("token")

		if userDiscordID == "" || token == "" {
			http.Error(w, "Missing parameters", http.StatusBadRequest)
			return
		}

		// Load HTML content from file
		htmlContent, err := os.ReadFile("./static/verify.html")
		if err != nil {
			http.Error(w, "Error loading verification page", http.StatusInternalServerError)
			return
		}

		// Replace placeholders with actual values
		pageContent := strings.Replace(string(htmlContent), "{{USER_DISCORD_ID}}", userDiscordID, -1)
		pageContent = strings.Replace(pageContent, "{{TOKEN}}", token, -1)

		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(pageContent))
		return
	}
	// From here on, handle POST requests
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ApiResponse{Success: false, Message: "Method not allowed"})
		return
	}

	userDiscordID := r.FormValue("user-discord-id")
	token := r.FormValue("token")

	if userDiscordID == "" || token == "" {
		json.NewEncoder(w).Encode(ApiResponse{Success: false, Message: "Missing parameters"})
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	mutex.Lock()
	file, err := os.ReadFile("/data/tokens.json")
	if err != nil {
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}

	var tokens map[string]TokenData
	err = json.Unmarshal(file, &tokens)
	if err != nil {
		http.Error(w, "Error parsing file", http.StatusInternalServerError)
		return
	}
	mutex.Unlock()

	if tokenData, ok := tokens[userDiscordID]; ok {
		if tokenData.Token == token {
			log.Println("Verification successful")
			json.NewEncoder(w).Encode(ApiResponse{Success: true, Message: "Verification successful"})

			// add role to user
			log.Printf("Adding role for -\n"+
				"   User ID: %s\n"+
				"   Guild ID: %s\n"+
				"   Role ID: %s\n",
				userDiscordID, tokens[userDiscordID].GuildID, os.Getenv("AUTH_ROLE_ID"))

			mutex.Lock()
			err := session.GuildMemberRoleAdd(tokens[userDiscordID].GuildID, userDiscordID, os.Getenv("AUTH_ROLE_ID"))
			if err != nil {
				log.Println("Error adding roll to user:", err)
			}
			log.Println("Role added successfully")
			mutex.Unlock()

			// remove key from map
			delete(tokens, userDiscordID)

			// Write the updated tokens back to the file
			updatedData, err := json.Marshal(tokens)
			if err != nil {
				http.Error(w, "Error marshalling JSON", http.StatusInternalServerError)
			}

			mutex.Lock()
			err = os.WriteFile("/data/tokens.json", updatedData, 0644)
			mutex.Unlock()

			if err != nil {
				http.Error(w, "Error writing to file", http.StatusInternalServerError)
			}
		} else {
			json.NewEncoder(w).Encode(ApiResponse{Success: false, Message: "Invalid token"})
			w.WriteHeader(http.StatusUnauthorized)
		}
	} else {
		http.Error(w, "User not found", http.StatusNotFound)
	}
}

func StartServer(sessionPass *discordgo.Session) {
	session = sessionPass
	port := os.Getenv("PORT")
	http.HandleFunc("/generate-user-token", GenerateUserTokenHandler)
	http.HandleFunc("/verify", VerifyHandler)

	fmt.Println("Server is running on port", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
