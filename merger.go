package submitqueue

import (
	"time"
)

type TestResult struct {
	Passed bool
	Error  string
}

type RunningTest interface {
	RepoSha1() string
	PRSha1() string
	Result() <-chan (TestResult)
}

type Project interface {
	Test(PullRequest) RunningTest
}

type Repo interface {
	IsGithub() bool
	HeadSha1() string
}

type PullRequest interface {
	HeadSha1() string
	IsMergeCandidate() (bool, string)
	Merge()
}

type Reporter struct{}

func (r *Reporter) Report(req SubmitRequest, msg string, args ...interface{}) {

}

func processQueue(queue *SubmitQueue, reporter *Reporter) {
	for {
		if !queue.IsSorted() {
			queue.Sort()
		}

		req, err := queue.Dequeue()
		if err != nil {
			if err != ErrQueueEmpty {
				panic("error pulling from queue")
			}
			time.Sleep(time.Second * 5)
		}

		processRequest(req, reporter, queue)
	}
}

func processRequest(req SubmitRequest, reporter *Reporter, queue *SubmitQueue) {
	if req.GetRepo().IsGithub() && req.GetPR().HeadSha1() != req.Sha1() {
		reporter.Report(req, "PR updated; re-queuing")
		return
	}

	if ok, reason := req.GetPR().IsMergeCandidate(); !ok {
		reporter.Report(req, "unable to automatically merge pr: %v", reason)
		return
	}

	test := req.GetProject().Test(req.GetPR())
	result := <-test.Result()
	if !result.Passed {
		reporter.Report(req, "failed to automatically merge; error running tests: %s", result.Error)
		return
	}

	if req.GetRepo().HeadSha1() != test.RepoSha1() || req.GetPR().HeadSha1() != test.PRSha1() {
		reporter.Report(req, "PR changed while being tested and has been re-scheduled")
		queue.Enqueue(req)
		return
	}

	req.GetPR().Merge()
	reporter.Report(req, "pr automatically merged!")
}
