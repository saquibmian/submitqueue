package submitqueue

import (
	"time"
)

type TestResult struct {
	Passed bool
	Error  string
}

type RunningTest struct {
	Result <-chan (TestResult)
}

type Project interface {
	Test(PullRequest) (RunningTest, error)
}

type Repo interface {
	IsGithub() bool
	HeadSha1() (string, error)
}

type PullRequest interface {
	HeadSha1() (string, error)
	IsMergeCandidate() (bool, string, error)
	Merge() error
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
	repo := req.GetRepo()
	pr := req.GetPR()

	// make sure PR is mergeable
	repoHeadSha1, err := repo.HeadSha1()
	if err != nil {
		queue.Enqueue(req)
		reporter.Report(req, "unable to fetch repo's SHA1; requeued")
		return
	}
	prHeadSha1, err := pr.HeadSha1()
	if err != nil {
		queue.Enqueue(req)
		reporter.Report(req, "unable to fetch PR's SHA1; requeued")
		return
	}
	if repo.IsGithub() && prHeadSha1 != req.Sha1() {
		reporter.Report(req, "PR updated; re-queue when ready")
		return
	}
	if ok, reason, err := pr.IsMergeCandidate(); err != nil {
		queue.Enqueue(req)
		reporter.Report(req, "unable to determine if PR is mergable; requeued")
		return
	} else if !ok {
		reporter.Report(req, "unable to automatically merge pr: %v", reason)
		return
	}

	// run tests
	test, err := req.GetProject().Test(pr)
	if err != nil {
		queue.Enqueue(req)
		reporter.Report(req, "error requesting tests; requeued: %v", err)
		return
	}
	result := <-test.Result // todo timeout here
	if !result.Passed {
		reporter.Report(req, "failed to automatically merge; tests failed: %s", result.Error)
		return
	}

	// make sure head hasn't changed
	currentRepoHeadSha1, err := repo.HeadSha1()
	if err != nil {
		queue.Enqueue(req)
		reporter.Report(req, "unable to fetch repo's SHA1; requeued")
		return
	}
	currentPrHeadSha1, err := pr.HeadSha1()
	if err != nil {
		queue.Enqueue(req)
		reporter.Report(req, "unable to fetch PR's SHA1; requeued")
		return
	}
	if currentRepoHeadSha1 != repoHeadSha1 || currentPrHeadSha1 != prHeadSha1 {
		reporter.Report(req, "PR changed while being tested and has been re-scheduled")
		queue.Enqueue(req)
		return
	}

	pr.Merge()
	reporter.Report(req, "pr automatically merged!")
}
