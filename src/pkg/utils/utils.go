package utils

import (
	"encoding/json"
	"github.com/bwmarrin/discordgo"
	"io/ioutil"
	"log"
	"os"
	"utk-auth-go/src/pkg/canvas"
)

type ServerConfig struct {
	Courses []canvas.Course `json:"courses"`
}

func GuildIdExists(guildId string) bool {
	file, err := ioutil.ReadFile("data/server_config.json")
	if err != nil {
		log.Println("Error reading server_config.json while checking for guildId")
		return false
	}
	var serverConfig ServerConfig
	err = json.Unmarshal(file, &serverConfig)
	if err != nil {
		log.Println("Error unmarshalling server_config.json while checking for guildId")
		return false
	}
	for _, course := range serverConfig.Courses {
		if course.GuildId == guildId {
			return true
		}
	}
	return false
}

var (
	// name that the command is invoked by
	RegisterCourseName = "registercourse"

	// invoked by "/registercourse [canvas_secret]"
	RegisterCourseCommand = discordgo.ApplicationCommand{
		Name:        "registercourse",
		Description: "Register your course to the current Discord server using your Canvas API secret",

		Type: discordgo.ChatApplicationCommand,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "canvas_secret",
				Description: "Your Canvas API Secret",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "course_id",
				Description: "Your Canvas course ID",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "auth_role_id",
				Description: "Your student authenticated role ID",
				Required:    true,
			},
		},
	}
)

func RegisterCourse(guildId string, canvasSecret string, courseId string, authRoleId string) error {
  log.Println("Registering course for guildId:", guildId)
	if _, err := os.Stat("data/server_config.json"); os.IsNotExist(err) {
		_, err := os.Create("data/server_config.json")
		if err != nil {
			log.Println("Error creating server_config.json while registering course")
			return err
		}
	}

	// open /data/server_config.json and add a new course to the list
	file, err := ioutil.ReadFile("data/server_config.json")
	if err != nil {
		log.Println("Error reading server_config.json while registering course")
	}
	var serverConfig ServerConfig
	err = json.Unmarshal(file, &serverConfig)
	if err != nil {
		log.Println("Error unmarshalling server_config.json while registering course")
		return err
	}

	students, err := canvas.GetCourseStudents(guildId, canvasSecret)
	if err != nil {
		log.Println("Error getting course students while registering course")
		return err
	}
	newCourse := canvas.Course{
		GuildId:      guildId,
		CanvasSecret: canvasSecret,
		CourseId:     courseId,
		Students:     students,
		AuthRoleId:   authRoleId,
	}
	serverConfig.Courses = append(serverConfig.Courses, newCourse)
	serverConfigBytes, err := json.Marshal(serverConfig)
	if err != nil {
		log.Println("Error marshalling server_config.json while registering course")
		return err
	}
	err = ioutil.WriteFile("data/server_config.json", serverConfigBytes, 0644)
	if err != nil {
		log.Println("Error writing server_config.json while registering course")
		return err
	}
	return nil
}
