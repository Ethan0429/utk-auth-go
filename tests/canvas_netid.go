package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

// Enrollment represents the structure of the enrollment data in the JSON response
type Enrollment struct {
	User struct {
		LoginID string `json:"login_id"`
		Name    string `json:"name"`
	} `json:"user"`
}

func getNetIds() (map[string]string, []Enrollment, error) {
	canvasApiToken := os.Getenv("CANVAS_API_TOKEN")
	courseId := os.Getenv("COURSE_ID")

	url := fmt.Sprintf("https://canvas.instructure.com/api/v1/courses/%s/enrollments?type=StudentEnrollment&per_page=230", courseId)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}

	request.Header.Add("Authorization", "Bearer "+canvasApiToken)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, nil, err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, nil, err
	}

	var enrollments []Enrollment
	err = json.Unmarshal(body, &enrollments)
	if err != nil {
		return nil, nil, err
	}

	var netIds map[string]string = make(map[string]string)
	for _, enrollment := range enrollments {
		words := strings.Fields(enrollment.User.Name)
		netIds[enrollment.User.LoginID] = words[0] + " " + words[len(words)-1]
	}

	return netIds, enrollments, nil
}

func main() {
	netIds, enrollments, err := getNetIds()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	count := 0
	for netId, name := range netIds {
		fmt.Println(count, ":", netId, "-", name)
		count++
	}

	// marshal enrollment data to json
	enrollmentData, err := json.MarshalIndent(enrollments, "", "  ")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// write to file
	err = ioutil.WriteFile("enrollments.json", enrollmentData, 0644)

	// marshal net id map to json
	netIdData, err := json.MarshalIndent(netIds, "", "  ")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	err = ioutil.WriteFile("netids.json", netIdData, 0644)
}
