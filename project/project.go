package project

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"text/template"

	"github.com/saquibmian/submitqueue/scm"
)

var (
	projectFile        = "projects.json"
	projects           map[string]Project
	ErrProjectNotFound = errors.New("project not found")
)

// LoadProjects loads all projects defined in project.json
func LoadProjects() error {
	projects = make(map[string]Project)
	file, err := os.Open(projectFile)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(file)
	configs := []projectConfig{}
	if err = decoder.Decode(&configs); err != nil {
		return err
	}
	for _, config := range configs {
		projects[config.Name] = &project{config, NewQueue()}
	}
	if len(projects) == 0 {
		return errors.New("no projects defined")
	}
	return nil
}

// Get gets a loaded project by name
func Get(name string) (Project, error) {
	proj, ok := projects[name]
	if !ok {
		return nil, ErrProjectNotFound
	}
	return proj, nil
}

// Projects returns all loaded projects
func Projects() map[string]Project {
	// todo don't expose map here
	return projects
}

type testResult struct {
	Passed bool
	Error  error
}

type RunningTest struct {
	result <-chan testResult
}

func (t *RunningTest) Wait() (bool, error) {
	result := <-t.result
	return result.Passed, result.Error
}

type Project interface {
	Name() string
	Queue() *SubmitQueue
	GetRepo(name string) (scm.Repo, error)
	GetPR(repo string, number int) (scm.PullRequest, error)
	Test(SubmitRequest) (RunningTest, error)
}

type scmConfig struct {
	Type        string `json:"type"`
	ProjectName string `json:"projectName"`
	Server      string `json:"server,omitempty"`
}

type testRequestConfig struct {
	URL          string            `json:"url"`
	Method       string            `json:"method,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	BodyTemplate string            `json:"body"`
}

type projectConfig struct {
	Name       string            `json:"name"`
	ScmConfig  scmConfig         `json:"scm"`
	TestConfig testRequestConfig `json:"testConfig"`
}

type project struct {
	config projectConfig
	queue  *SubmitQueue
}

func (p *project) Name() string {
	return p.config.Name
}

func (p *project) Queue() *SubmitQueue {
	return p.queue
}

func (p *project) GetRepo(repo string) (scm.Repo, error) {
	if p.config.ScmConfig.Type == "github" {
		return scm.NewGithubRepo(p.config.ScmConfig.ProjectName, repo), nil
	}
	return nil, nil
}

func (p *project) GetPR(repo string, number int) (scm.PullRequest, error) {
	if p.config.ScmConfig.Type == "github" {
		return scm.NewGithubPullRequest(p.config.ScmConfig.ProjectName, repo, number), nil
	}
	return nil, nil
}

func (p *project) Test(testRequest SubmitRequest) (RunningTest, error) {
	tc := p.config.TestConfig
	tmpl, err := template.New(tc.Method + tc.URL).Parse(tc.BodyTemplate)
	if err != nil {
		return RunningTest{}, err
	}
	body := new(bytes.Buffer)
	if err = tmpl.Execute(body, testRequest); err != nil {
		return RunningTest{}, err
	}
	method := tc.Method
	if method == "" {
		method = "POST"
	}
	req, err := http.NewRequest(method, tc.URL, body)
	if err != nil {
		return RunningTest{}, err
	}
	for header, value := range tc.Headers {
		req.Header.Set(header, value)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return RunningTest{}, err
	}
	defer resp.Body.Close()

	// todo check response status code

	// todo do result properly
	resultChannel := make(chan testResult, 1)
	resultChannel <- testResult{true, nil}

	return RunningTest{resultChannel}, err
}
