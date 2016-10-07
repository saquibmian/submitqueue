package project

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"text/template"

	"github.com/saquibmian/submitqueue/scm"
)

var (
	projects []Project
)

func init() {
	file, err := os.Open("projects.json")
	if err != nil {
		panic("error reading projects.json")
	}
	decoder := json.NewDecoder(file)
	configs := []projectConfig{}
	if err = decoder.Decode(&configs); err != nil {
		panic("error parsing projects.json")
	}
	for _, config := range configs {
		projects = append(projects, &project{config})
	}
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
	Test(scm.PullRequest) (RunningTest, error)
}

func Get(name string) (Project, error) {
	return nil, nil
}

func Projects() []Project {
	return nil
}

type testRequestConfig struct {
	Url          string            `json:"url"`
	Method       string            `json:"method"`
	Headers      map[string]string `json:"headers"`
	BodyTemplate string            `json:"body"`
}

func (c testRequestConfig) GetBodyTemplate() (*template.Template, error) {
	return template.New(c.Method + c.Url).Parse(c.BodyTemplate)
}

type projectConfig struct {
	Name       string            `json:"name"`
	TestConfig testRequestConfig `json:"testConfig"`
}

type project struct {
	config projectConfig
}

func (p *project) Name() string {
	return p.config.Name
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
	req, err := http.NewRequest(tc.Method, tc.Url, nil)
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
