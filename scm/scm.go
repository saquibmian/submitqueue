package scm

type Repo interface {
	IsGithub() bool
	HeadSha1() (string, error)
}

type PullRequest interface {
	HeadSha1() (string, error)
	IsMergeCandidate() (bool, string, error)
	Merge() error
}
