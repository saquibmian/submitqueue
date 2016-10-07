package github

import (
	"os"

	"golang.org/x/oauth2"

	gh "github.com/google/go-github/github"
	"github.com/saquibmian/submitqueue/scm"
)

var (
	masterBranchName = "maser"
	accessToken      = os.Getenv("GITHUB_ACCESS_TOKEN")
)

func getClient() *gh.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	return gh.NewClient(tc)
}

type githubRepo struct {
	owner    string
	repoName string
	client   *gh.Client
}

func (r *githubRepo) IsGithub() bool {
	return true
}
func (r *githubRepo) HeadSha1() (string, error) {
	branch, _, err := r.client.Repositories.GetBranch(r.owner, r.repoName, masterBranchName)
	return *branch.Commit.SHA, err
}

func NewRepo(owner string, repo string) scm.Repo {
	return &githubRepo{
		owner,
		repo,
		getClient(),
	}
}

type githubPullRequest struct {
	owner    string
	repoName string
	number   int
	client   *gh.Client
}

func (p *githubPullRequest) HeadSha1() (string, error) {
	pr, _, err := p.client.PullRequests.Get(p.owner, p.repoName, p.number)
	return *pr.Head.SHA, err
}

func (p *githubPullRequest) IsMergeCandidate() (bool, string, error) {
	pr, _, err := p.client.PullRequests.Get(p.owner, p.repoName, p.number)
	// todo: reason
	return *pr.Mergeable, "", err
}

func (p *githubPullRequest) Merge() error {
	_, _, err := p.client.PullRequests.Merge(p.owner, p.repoName, p.number, "Auto-merge by submitqueue", nil)
	return err
}

func NewPullRequest(owner string, repo string, number int) scm.PullRequest {
	return &githubPullRequest{
		owner,
		repo,
		number,
		getClient(),
	}
}
