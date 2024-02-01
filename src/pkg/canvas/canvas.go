package canvas

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
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
	var students []Student
	url := fmt.Sprintf("https://canvas.instructure.com/api/v1/courses/%s/enrollments?per_page=100", courseId)

	for url != "" {
		request, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Println("Error generating Canvas API request for GetCourseStudents:", err)
			return nil, err
		}

		request.Header.Add("Authorization", "Bearer "+canvasSecret)

		client := &http.Client{}
		response, err := client.Do(request)
		if err != nil {
			log.Println("Error sending request to Canvas API while getting course students:", err)
			return nil, err
		}
		defer response.Body.Close()

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Println("Error reading response from Canvas API while getting course students:", err)
			return nil, err
		}

		var enrollments []Enrollment
		err = json.Unmarshal(body, &enrollments)
		if err != nil {
			log.Println("Error unmarshalling response from Canvas API while getting course students:", err)
			return nil, err
		}

		for _, enrollment := range enrollments {
			words := strings.Fields(enrollment.User.Name)
			netId := enrollment.User.LoginID
			name := words[0] + " " + words[len(words)-1]
			students = append(students, Student{NetId: netId, Name: name})
		}

		// Find the URL for the next page
		links := response.Header["Link"]
		url = "" // Reset URL for the next iteration
		for _, link := range links {
			if strings.Contains(link, `rel="next"`) {
				parts := strings.Split(link, ";")
				if len(parts) > 0 {
					url = strings.Trim(parts[0], "<>")
					break
				}
			}
		}
	}

	return students, nil
}
