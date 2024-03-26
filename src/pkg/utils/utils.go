package utils

import (
	"encoding/json"
	"github.com/bwmarrin/discordgo"
	"log"
	"os"
	"sync"
	"utk-auth-go/src/pkg/canvas"
)

var mutex sync.Mutex

type ServerConfig struct {
	Courses []canvas.Course `json:"courses"`
}

// helper functions
func StrPtr(s string) *string {
	return &s
}

func NewEmbed(title string, description string, color int, fields []*discordgo.MessageEmbedField) *discordgo.MessageEmbed {
	if fields == nil {

		return &discordgo.MessageEmbed{
			Title:       title,
			Description: description,
			Color:       color,
		}
	}
	return &discordgo.MessageEmbed{
		Title:       title,
		Description: description,
		Color:       color,
		Fields:      fields,
	}
}

func NewEmbeds(embeds ...*discordgo.MessageEmbed) *[]*discordgo.MessageEmbed {
	return &embeds
}

func StudentExists(guildId string, netId string) (bool, error) {
	// open /data/server_config.json and check if student exists in any courses
	file, err := os.ReadFile("/data/server_config.json")
	if err != nil {
		log.Println("Error reading server_config.json while checking for student:", err)
		return false, err
	}

	if len(file) == 0 {
		log.Println("server_config.json is empty")
		return false, nil
	}

	var serverConfig ServerConfig
	err = json.Unmarshal(file, &serverConfig)
	if err != nil {
		log.Println("Error unmarshalling server_config.json while checking for student:", err)
		return false, err
	}

	for _, course := range serverConfig.Courses {
		if course.GuildId == guildId {
			for _, student := range course.Students {
				if student.NetId == netId {
					return true, nil
				}
			}
			log.Println("No Student found for guildId:", guildId)
		} else {
			log.Println("No courses found for guildId:", guildId)
		}
	}

	return false, nil
}

func GuildIdExists(guildId string) (bool, error) {
	file, err := os.ReadFile("/data/server_config.json")
	if err != nil {
		log.Println("Error reading server_config.json while checking for guildId:", err)
		return false, err
	}
	if len(file) == 0 {
		return false, nil
	}
	var serverConfig ServerConfig
	err = json.Unmarshal(file, &serverConfig)
	if err != nil {
		log.Println("Error unmarshalling server_config.json while checking for guildId:", err)
		return false, err
	}
	for _, course := range serverConfig.Courses {
		if course.GuildId == guildId {
			return true, nil
		}
	}
	return false, nil
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

	// open /data/server_config.json and add a new course to the list
	file, err := os.ReadFile("/data/server_config.json")
	if err != nil {
		log.Println("Error reading server_config.json while registering course:", err)
		return err
	}
	var serverConfig ServerConfig

	if len(file) != 0 {
		err = json.Unmarshal(file, &serverConfig)
		if err != nil {
			log.Println("Error unmarshalling server_config.json while registering course:", err)
			return err
		}
	} else {
		// populate server config with empty courses
		serverConfig = ServerConfig{
			Courses: []canvas.Course{},
		}
	}

	students, err := canvas.GetCourseStudents(os.Getenv("UTK_CANVAS_COURSE_ID_PREFIX")+courseId, canvasSecret)
	if err != nil {
		return err
	}
	newCourse := canvas.Course{
		GuildId:      guildId,
		CanvasSecret: canvasSecret,
		CourseId:     os.Getenv("UTK_CANVAS_COURSE_ID_PREFIX") + courseId,
		Students:     students,
		AuthRoleId:   authRoleId,
	}
	serverConfig.Courses = append(serverConfig.Courses, newCourse)
	serverConfigBytes, err := json.Marshal(serverConfig)
	if err != nil {
		log.Println("Error marshalling server_config.json while registering course")
		return err
	}
	err = os.WriteFile("/data/server_config.json", serverConfigBytes, 0644)
	if err != nil {
		log.Println("Error writing server_config.json while registering course")
		return err
	}
	return nil
}

func GetCourseObject(guildID string) (*canvas.Course, error) {
	mutex.Lock()
	defer mutex.Unlock()

	file, err := os.ReadFile("/data/server_config.json")
	if err != nil {
		log.Println("Error reading server_config.json while getting course object:", err)
		return nil, err
	}
	if len(file) == 0 {
		return nil, nil
	}
	var serverConfig ServerConfig
	err = json.Unmarshal(file, &serverConfig)
	if err != nil {
		log.Println("Error unmarshalling server_config.json while getting course object:", err)
		return nil, err
	}
	for _, course := range serverConfig.Courses {
		if course.GuildId == guildID {
			return &course, nil
		}
	}
	return nil, nil
}
