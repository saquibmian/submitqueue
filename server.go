package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/saquibmian/submitqueue/project"
	"net/http"
	"os"
	"time"
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
		q.Enqueue(project.SubmitRequest{
			Priority:    project.P1,
			IsEmergency: false,
			Project:     proj.Name(),
			Repo:        "submitqueue",
			PRNumber:    1,
			FromRef:     "ref/heads/random",
			Sha1:        "",
		})
		q.Enqueue(project.SubmitRequest{
			Priority:    project.P1,
			IsEmergency: false,
			Project:     proj.Name(),
			Repo:        "submitqueue",
			PRNumber:    2,
			FromRef:     "ref/heads/random",
			Sha1:        "7ddcbfafe70f940a0bdd6a663d95bba08775d85b",
		})
		q.Sort()

		fmt.Printf("=== queue for project %s ===\n", proj.Name())
		q.Dump()

		processQueue(proj, nil)
	}

	http.ListenAndServe(":8080", nil)

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
			continue
		}

		processRequest(req, reporter, proj)
		time.Sleep(time.Second * 1)
	}
}

func processRequest(req project.SubmitRequest, reporter *Reporter, proj project.Project) {
	logger := log.WithFields(log.Fields{
		"proj": req.Project,
		"repo": req.Repo,
		"pr":   req.PRNumber,
	})

	repo, err := proj.GetRepo(req.Repo)
	if err != nil {
		logger.Error(err)
		reporter.Report(req, "error trying to process merge")
		return
	}
	pr, err := proj.GetPR(req.Repo, req.PRNumber)
	if err != nil {
		logger.Error(err)
		reporter.Report(req, "error trying to process merge")
		return
	}

	// make sure PR is mergeable
	repoHeadSha1, err := repo.HeadSha1()
	if err != nil {
		logger.Error(err)
		proj.Queue().Enqueue(req)
		reporter.Report(req, "unable to fetch repo's SHA1; requeued")
		return
	}
	prHeadSha1, err := pr.HeadSha1()
	if err != nil {
		logger.Error(err)
		proj.Queue().Enqueue(req)
		reporter.Report(req, "unable to fetch PR's SHA1; requeued")
		return
	}
	// todo kill this
	if repo.IsGithub() && prHeadSha1 != req.Sha1 {
		logger.Infof("pr updated; expected %s, found %s", req.Sha1, prHeadSha1)
		reporter.Report(req, "PR updated; re-queue when ready")
		return
	}
	if ok, reason, err := pr.IsMergeCandidate(); err != nil {
		logger.Error(err)
		proj.Queue().Enqueue(req)
		reporter.Report(req, "unable to determine if PR is mergable; requeued")
		return
	} else if !ok {
		reporter.Report(req, "unable to automatically merge pr: %v", reason)
		return
	}

	// run tests
	test, err := proj.Test(req)
	if err != nil {
		logger.Error(err)
		proj.Queue().Enqueue(req)
		reporter.Report(req, "error requesting tests; requeued: %v", err)
		return
	}
	passed, err := test.Wait()
	if !passed {
		logger.Error(err)
		reporter.Report(req, "failed to automatically merge; tests failed: %s", err.Error())
		return
	}

	// make sure head hasn't changed
	currentRepoHeadSha1, err := repo.HeadSha1()
	if err != nil {
		logger.Error(err)
		proj.Queue().Enqueue(req)
		reporter.Report(req, "unable to fetch repo's SHA1; requeued")
		return
	}
	currentPrHeadSha1, err := pr.HeadSha1()
	if err != nil {
		logger.Error(err)
		proj.Queue().Enqueue(req)
		reporter.Report(req, "unable to fetch PR's SHA1; requeued")
		return
	}
	if currentRepoHeadSha1 != repoHeadSha1 || currentPrHeadSha1 != prHeadSha1 {
		reporter.Report(req, "PR changed while being tested and has been re-scheduled")
		proj.Queue().Enqueue(req)
		return
	}

	err = pr.Merge()
	if err != nil {
		logger.Error(err)
		reporter.Report(req, "error when trying to merge PR")
		return
	}
	reporter.Report(req, "pr automatically merged!")
}

type Reporter struct{}

func (r *Reporter) Report(req project.SubmitRequest, msg string, args ...interface{}) {
	fmt.Println(msg)
}
