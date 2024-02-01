package canvas

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type Student struct {
	NetId string `json:"netId"`
	Name  string `json:"name"`
}

type Course struct {
	GuildId      string    `json:"guildId"`
	CanvasSecret string    `json:"canvasSecret"`
	CourseId     string    `json:"courseId"`
	Students     []Student `json:"students"`
  AuthRoleId   string    `json:"authRoleId"`
}

// Enrollment represents the structure of the enrollment data in the JSON response
type Enrollment struct {
	User struct {
		LoginID string `json:"login_id"`
		Name    string `json:"name"`
	} `json:"user"`
}

func GetCourseStudents(courseId string, canvasSecret string) ([]Student, error) {
	url := fmt.Sprintf("https://canvas.instructure.com/api/v1/courses/%s/enrollments?type=StudentEnrollment&per_page=230", courseId)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Add("Authorization", "Bearer "+canvasSecret)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var enrollments []Enrollment
	err = json.Unmarshal(body, &enrollments)
	if err != nil {
		return nil, err
	}

	var netIds map[string]string = make(map[string]string)
	for _, enrollment := range enrollments {
		words := strings.Fields(enrollment.User.Name)
		netIds[enrollment.User.LoginID] = words[0] + " " + words[len(words)-1]
	}

	// create list of students from netIds
	var students []Student
	for netId, name := range netIds {
		students = append(students, Student{NetId: netId, Name: name})
	}
	return students, nil
}
