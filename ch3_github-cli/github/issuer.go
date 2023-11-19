package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	githubIssueUrl = "https://api.github.com/repos/%s/%s/issues"
	issueRegex     = "^Title: ((?s).*)\nBody: ((?s).*)\nMilestone: ((?s).*)\nLabels: ((?s).*)\nAssignees: ((?s).*)\n$"
)

type GetIssuesRes struct {
	TotalCount int `json:"total_count"`
	Items      []CreateIssueReq
}

type User struct {
	Login   string
	HTMLURL string `json:"html_url"`
}

type GetIssueRes struct {
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Milestone *Milestone `json:"milestone"`
	Labels    []Label    `json:"labels"`
	Assignees []Assignee `json:"assignees"`
}

type Milestone struct {
	Number int `json:"number"`
}

type Label struct {
	Name string `json:"name"`
}

type Assignee struct {
	Login string `json:"login"`
}

type CreateIssueReq struct {
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Milestone string   `json:"milestone,omitempty"`
	Labels    []string `json:"labels,omitempty"`
	Assignees []string `json:"assignees,omitempty"`
}

var Owner string
var Repo string

func GetIssues() error {
	repoUrl := fmt.Sprintf(githubIssueUrl, Owner, Repo)
	httpGet, _ := http.NewRequest(http.MethodGet, repoUrl, nil)
	httpGet.Header.Set("Authorization", "Bearer "+os.Getenv("GITHUB_AUTH_TOKEN"))
	httpGet.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{}).Do(httpGet)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	issues := new([]GetIssueRes)
	err = json.NewDecoder(resp.Body).Decode(issues)
	if err != nil {
		return err
	}
	t := table.NewWriter()
	t.AppendHeader(table.Row{"Number", "Title"})
	for _, iss := range *issues {
		t.AppendRow(table.Row{iss.Number, iss.Title})
	}
	fmt.Println(t.Render())
	return nil
}

func getIssue(id int) (*GetIssueRes, error) {
	repoUrl := fmt.Sprintf(githubIssueUrl+"/%d", Owner, Repo, id)
	req, err := http.NewRequest(http.MethodGet, repoUrl, nil)
	if err != nil {
		return nil, err
	}
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var issue GetIssueRes
	err = json.NewDecoder(resp.Body).Decode(&issue)
	if err != nil {
		return nil, err
	}
	return &issue, nil
}

func GetIssue(id int) error {
	iss, err := getIssue(id)
	if err != nil {
		return err
	}
	t := table.NewWriter()
	t.AppendHeader(table.Row{"Number", "Title", "Milestone", "Labels", "Assignees"})
	t.AppendRow(table.Row{iss.Number, iss.Title, *iss.Milestone, iss.Labels, iss.Assignees})
	fmt.Println(t.Render())
	return nil
}

func CreateIssue() error {
	filename, err := prepareTempFile(map[string]string{})
	if err != nil {
		return err
	}
	err = openTempFile(filename)
	if err != nil {
		return err
	}
	content, err := grabFromTextEditor(filename)
	if err != nil {
		return err
	}
	req := parseUserInput(content)
	repoUrl := fmt.Sprintf(githubIssueUrl, Owner, Repo)
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	//fmt.Printf("request body: %s\n", data)
	httpPost, _ := http.NewRequest(http.MethodPost, repoUrl, bytes.NewBuffer(data))
	httpPost.Header.Set("Authorization", "Bearer "+os.Getenv("GITHUB_AUTH_TOKEN"))
	httpPost.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{}).Do(httpPost)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected response ststus: %d\n response body: %s\n", resp.StatusCode, respBody)
	}
	fmt.Println("CreateIssueReq successfully created")
	return nil
}

func EditIssue(id int) error {
	issue, err := getIssue(id)
	labels := make([]string, 0)
	for _, l := range issue.Labels {
		labels = append(labels, l.Name)
	}
	assignees := make([]string, 0)
	for _, a := range issue.Assignees {
		assignees = append(assignees, a.Login)
	}
	issueInfo := map[string]string{
		"Title":     issue.Title,
		"Body":      issue.Body,
		"Milestone": strconv.Itoa(issue.Milestone.Number),
		"Labels":    strings.Join(labels, ","),
		"Assignees": strings.Join(assignees, ","),
	}
	filename, err := prepareTempFile(issueInfo)
	defer os.Remove(filename)
	if err != nil {
		return err
	}
	err = openTempFile(filename)
	if err != nil {
		return err
	}
	content, err := grabFromTextEditor(filename)
	if err != nil {
		return err
	}
	iss := parseUserInput(content)
	data, err := json.Marshal(iss)
	if err != nil {
		return err
	}
	repoUrl := fmt.Sprintf(githubIssueUrl+"/%d", Owner, Repo, id)
	httpPatch, _ := http.NewRequest(http.MethodPatch, repoUrl, bytes.NewBuffer(data))
	httpPatch.Header.Set("Authorization", "Bearer "+os.Getenv("GITHUB_AUTH_TOKEN"))
	httpPatch.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{}).Do(httpPatch)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected response ststus: %d\n response body: %s\n", resp.StatusCode, respBody)
	}
	fmt.Println("CreateIssueReq successfully patched")
	return nil
}

func openTempFile(filename string) error {
	userEditor := os.Getenv("EDITOR")
	if userEditor == "" {
		userEditor = "notepad"
	}
	cmd := exec.Command(userEditor, filename)
	err := cmd.Run()
	return err
}

func grabFromTextEditor(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

func fillTempFile(file *os.File, data map[string]string) error {
	var err error
	_, err = file.WriteString(fmt.Sprintf("Title: %s\n", data["Title"]))
	_, err = file.WriteString(fmt.Sprintf("Body: %s\n", data["Body"]))
	_, err = file.WriteString(fmt.Sprintf("Milestone: %s\n", data["Milestone"]))
	_, err = file.WriteString(fmt.Sprintf("Labels: %s\n", data["Labels"]))
	_, err = file.WriteString(fmt.Sprintf("Assignees: %s\n", data["Assignees"]))
	return err
}

func prepareTempFile(data map[string]string) (string, error) {
	file, err := os.CreateTemp(os.TempDir(), "*")
	if err != nil {
		return "", err
	}
	err = fillTempFile(file, data)
	if err != nil {
		return "", err
	}
	err = file.Close()
	if err != nil {
		return "", err
	}
	return file.Name(), nil
}
func parseUserInput(content []byte) *CreateIssueReq {
	input := string(content)
	req := new(CreateIssueReq)
	r := regexp.MustCompile(issueRegex)
	m := r.FindStringSubmatch(input)
	if m[1] == "" {
		log.Fatal("Title must be set")
	}
	req.Title = m[1]
	if m[2] != "" {
		req.Body = m[2]
	}
	if m[3] != "" {
		req.Milestone = m[3]
	}
	if m[4] != "" {
		req.Labels = strings.Split(m[4], ",")
	}
	if m[5] != "" {
		req.Assignees = strings.Split(m[5], ",")
	}
	return req
}
