package registercourse

import (
	"encoding/json"
	"io/ioutil"
	"utk-auth-go/src/pkg/common"

	"github.com/bwmarrin/discordgo"
)

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

func RegisterCourse(guildId string, courseId string, authRoleId string) error {
	// open /data/server_config.json and add a new course to the list
	file, err := ioutil.ReadFile("/data/server_config.json")
	if err != nil {
		return err
	}
	var serverConfig map[string]interface{}

	if len(file) != 0 {
		err = json.Unmarshal(file, &serverConfig)
		if err != nil {
			return err
		} else {
			serverConfig = make(map[string]interface{})
		}
	}

	serverConfig[guildId] = map[string]string{
		"courseId":   common.UtkCanvasCourseIDPrefix + courseId,
		"authRoleId": authRoleId,
	}

	return nil
}
