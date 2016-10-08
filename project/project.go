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
	projects = make(map[string]Project)
)

func LoadProjects() error {
	file, err := os.Open("projects.json")
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

func Get(name string) (Project, error) {
	return nil, nil
}

func Projects() map[string]Project {
	return projects
}

type TestResult struct {
	Passed bool
	Error  string
}

type RunningTest struct {
	Result <-chan TestResult
}

type Project interface {
	Name() string
	Queue() *SubmitQueue
	GetRepo(name string) scm.Repo
	GetPR(repo string, number int) scm.PullRequest
	Test(scm.PullRequest) (RunningTest, error)
}

type testRequestConfig struct {
	URL          string            `json:"url"`
	Method       string            `json:"method"`
	Headers      map[string]string `json:"headers"`
	BodyTemplate string            `json:"body"`
}

func (c testRequestConfig) GetBodyTemplate() (*template.Template, error) {
	return template.New(c.Method + c.URL).Parse(c.BodyTemplate)
}

type projectConfig struct {
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	TestConfig testRequestConfig `json:"testConfig"`
}

type project struct {
	config projectConfig
	queue *SubmitQueue
}

func (p *project) Name() string {
	return p.config.Name
}

func (p *project) Queue() *SubmitQueue {
	return p.queue
}

func (p *project) GetRepo(name string) scm.Repo {
	return nil
}

func (p *project) GetPR(repo string, number int) scm.PullRequest {
	return nil
}


func (p *project) Test(pr scm.PullRequest) (RunningTest, error) {
	tc := p.config.TestConfig
	tmpl, err := tc.GetBodyTemplate()
	if err != nil {
		return RunningTest{}, err
	}
	body := new(bytes.Buffer)
	if err = tmpl.Execute(body, nil); err != nil {
		return RunningTest{}, err
	}
	req, err := http.NewRequest(tc.Method, tc.URL, nil)
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
	resultChannel := make(chan TestResult, 1)
	resultChannel <- TestResult{true, ""}

	return RunningTest{resultChannel}, err
}
