package scm

import (
	"os"

	"golang.org/x/oauth2"

	"github.com/google/go-github/github"
)

var (
	masterBranchName = "master"
	accessToken      = os.Getenv("GITHUB_ACCESS_TOKEN")
)

func getGithubClient() *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	return github.NewClient(tc)
}

type githubRepo struct {
	owner    string
	repoName string
	client   *github.Client
}

func (r *githubRepo) IsGithub() bool {
	return true
}
func (r *githubRepo) HeadSha1() (string, error) {
	branch, _, err := r.client.Repositories.GetBranch(r.owner, r.repoName, masterBranchName)
	if err != nil {
		return "", err
	}
	return *branch.Commit.SHA, nil
}

func NewGithubRepo(owner string, repo string) Repo {
	return &githubRepo{
		owner,
		repo,
		getGithubClient(),
	}
}

type githubPullRequest struct {
	owner    string
	repoName string
	number   int
	client   *github.Client
}

func (p *githubPullRequest) HeadSha1() (string, error) {
	pr, _, err := p.client.PullRequests.Get(p.owner, p.repoName, p.number)
	if err != nil {
		return "", err
	}
	return *pr.Head.SHA, err
}

func (p *githubPullRequest) IsMergeCandidate() (bool, string, error) {
	pr, _, err := p.client.PullRequests.Get(p.owner, p.repoName, p.number)
	if err != nil {
		return false, "", err
	}
	if pr.Mergeable == nil {
		return false, "unable to determine if mergable", nil
	}
	// todo: reason
	return *pr.Mergeable, "", err
}

func (p *githubPullRequest) Merge() error {
	return nil
	// _, _, err := p.client.PullRequests.Merge(p.owner, p.repoName, p.number, "Auto-merge by submitqueue", nil)
	// return err
}

func NewGithubPullRequest(owner string, repo string, number int) PullRequest {
	return &githubPullRequest{
		owner,
		repo,
		number,
		getGithubClient(),
	}
}
