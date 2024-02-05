package auth

import (
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"log"
	"utk-auth-go/src/pkg/canvas"
	"utk-auth-go/src/pkg/utils"
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
	// invoked by "/auth" 
	AuthCommand = discordgo.ApplicationCommand{
		Name:        "auth",
		Description: "Authenticate with your NetID as a student",

		Type: discordgo.ChatApplicationCommand,
	}
)

func IsEnrolled(enrollments []canvas.Enrollment, guildID string) (bool, error) {
	courses, err := utils.GetRegisteredCourses(guildID)
	if err != nil {
		return false, err
	}

	// find if any element in enrollments has a course ID from courses
	for _, enrollment := range enrollments {
		for _, course := range courses {
			if enrollment.CourseID == course.CanvasCourseID {
				return true, nil
			}
		}
	}

	return false, nil
}
