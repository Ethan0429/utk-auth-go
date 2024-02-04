package utils

import (
	"encoding/json"
	"github.com/bwmarrin/discordgo"
	"io/ioutil"
	"utk-auth-go/src/pkg/canvas"
)

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

func GetRegisteredCourses(guildID string) ([]canvas.Course, error) {
	// open /data/server_config.json return courses pertaining to the guildID

	file, err := ioutil.ReadFile("/data/server_config.json")
	if err != nil {
		return nil, err
	}
	var serverConfig map[string][]canvas.Course
	err = json.Unmarshal(file, &serverConfig)
	if err != nil {
		return nil, err
	}
  return serverConfig[guildID], nil
}

func GetAuthRoleID(guildID string) (string, error) {
  file, err := ioutil.ReadFile("/data/server_config.json")
  if err != nil {
    return "", err
  }
  var serverConfig map[string][]canvas.Course
  err = json.Unmarshal(file, &serverConfig)
  if err != nil {
    return "", err
  }
  return serverConfig[guildID][0].AuthRoleID, nil
}

func GuildIDExists(guildID string) bool {
  file, err := ioutil.ReadFile("/data/server_config.json")
  if err != nil {
    return false
  }
  var serverConfig map[string][]canvas.Course
  err = json.Unmarshal(file, &serverConfig)
  if err != nil {
    return false
  }
  _, ok := serverConfig[guildID]
  return ok
}

