package canvas

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

type Course struct {
	CanvasCourseID string `json:"canvasCourseID"`
	AuthRoleID     string `json:"authRoleID"`
}

type User struct {
	LoginID string `json:"login_id"`
	Name    string `json:"name"`
}

// Enrollment represents the structure of the enrollment data in the JSON response
type Enrollment struct {
	CourseID string `json:"course_id"`
	User     User   `json:"user"`
}

func GetUserEnrollments(canvasClientSecret string) []Enrollment {
	url := "https://utk.instructure.com/api/v1/users/self/enrollments"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)
		return nil
	}

	req.Header.Set("Authorization", "Bearer "+canvasClientSecret)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return nil
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return nil
	}

	var enrollments []Enrollment
	err = json.Unmarshal(body, &enrollments)
	if err != nil {
		log.Println(err)
		return nil
	}

	return enrollments
}
