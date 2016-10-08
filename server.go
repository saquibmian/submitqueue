package main

import (
    "fmt"
    "time"
    "os"
    "github.com/saquibmian/submitqueue/project"
)

func main() {
    if err := project.LoadProjects(); err != nil {
        panic(err)
    }

    for name := range project.Projects() {
        fmt.Printf("loaded project: %s\n", name)
    }

    for _, proj := range project.Projects() {
        q := proj.Queue()
        q.Dump()
        go processQueue(proj, nil)
    }

    fmt.Printf("done\n")
}

func processQueue(proj project.Project, reporter *Reporter) {
    queue := proj.Queue()
    fmt.Printf("began processing queue for project %s\n", proj.Name())
    for {
		if !queue.IsSorted() {
			queue.Sort()
		}

		req, err := queue.Dequeue()
		if err != nil {
			if err != project.ErrQueueEmpty {
				fmt.Fprintf(os.Stderr, "error pulling from queue: %s", err.Error())
			}
			time.Sleep(time.Second * 5)
		}

		processRequest(req, reporter, proj)
	}
}

func processRequest(req project.SubmitRequest, reporter *Reporter, proj project.Project) {
	repo := proj.GetRepo(req.Repo)
	pr := proj.GetPR(req.Repo, req.PRNumber)

	// make sure PR is mergeable
	repoHeadSha1, err := repo.HeadSha1()
	if err != nil {
		proj.Queue().Enqueue(req)
		reporter.Report(req, "unable to fetch repo's SHA1; requeued")
		return
	}
	prHeadSha1, err := pr.HeadSha1()
	if err != nil {
		proj.Queue().Enqueue(req)
		reporter.Report(req, "unable to fetch PR's SHA1; requeued")
		return
	}
	// todo kill this
	if repo.IsGithub() && prHeadSha1 != req.Sha1 {
		reporter.Report(req, "PR updated; re-queue when ready")
		return
	}
	if ok, reason, err := pr.IsMergeCandidate(); err != nil {
		proj.Queue().Enqueue(req)
		reporter.Report(req, "unable to determine if PR is mergable; requeued")
		return
	} else if !ok {
		reporter.Report(req, "unable to automatically merge pr: %v", reason)
		return
	}

	// run tests
	test, err := proj.Test(pr)
	if err != nil {
		proj.Queue().Enqueue(req)
		reporter.Report(req, "error requesting tests; requeued: %v", err)
		return
	}
	passed, err := test.Wait()
	if !passed {
		reporter.Report(req, "failed to automatically merge; tests failed: %s", err.Error())
		return
	}

	// make sure head hasn't changed
	currentRepoHeadSha1, err := repo.HeadSha1()
	if err != nil {
		proj.Queue().Enqueue(req)
		reporter.Report(req, "unable to fetch repo's SHA1; requeued")
		return
	}
	currentPrHeadSha1, err := pr.HeadSha1()
	if err != nil {
		proj.Queue().Enqueue(req)
		reporter.Report(req, "unable to fetch PR's SHA1; requeued")
		return
	}
	if currentRepoHeadSha1 != repoHeadSha1 || currentPrHeadSha1 != prHeadSha1 {
		reporter.Report(req, "PR changed while being tested and has been re-scheduled")
		proj.Queue().Enqueue(req)
		return
	}

	pr.Merge()
	reporter.Report(req, "pr automatically merged!")
}

type Reporter struct{}

func (r *Reporter) Report(req project.SubmitRequest, msg string, args ...interface{}) {
    fmt.Printf(msg, args)
}
