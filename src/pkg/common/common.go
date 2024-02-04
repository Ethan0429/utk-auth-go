package common

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

func init() {
	// load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}
}

var (
	AuthRoleID             = os.Getenv("AUTH_ROLE_ID")
	CanvasClientID          = os.Getenv("CANVAS_CLIENT_ID")
	AuthServerUrl           = os.Getenv("AUTH_SEVER_URL")
	CanvasClientSecret      = os.Getenv("CANVAS_CLIENT_SECRET")
	CanvasInstallUrl        = os.Getenv("CANVAS_INSTALL_URL")
	UtkCanvasCourseIDPrefix = os.Getenv("UTK_CANVAS_COURSE_ID_PREFIX")
)
