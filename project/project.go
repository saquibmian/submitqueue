package project

import (
	"github.com/saquibmian/submitqueue/scm"
)

type TestResult struct {
	Passed bool
	Error  string
}

type RunningTest struct {
	Result <-chan (TestResult)
}

type Project interface {
	Test(scm.PullRequest) (RunningTest, error)
}
