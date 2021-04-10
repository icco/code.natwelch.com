package code

import (
	"fmt"
	"time"

	"gorm.io/gorm"
  "github.com/google/go-github/v34/github"
)

type Commit struct {
	gorm.Model

	Repo      string    `gorm:"index:idx_commit,unique",json:"repo"`
	User      string    `json:"user"`
	SHA       string    `gorm:"index:idx_commit,unique",json:"sha"`
	CreatedOn time.Time `json:"created_on"`
}

// String returns a string representation of a Commit.
func (c *Commit) String() string {
	return fmt.Sprintf("%s/%s#%s", c.User, c.Repo, c.SHA)
}

func UserRepos(client *github.Client, user string) ([]*github.Repository, error) {
  repos = client.repos(user_name).map { |r| r["full_name"].split("/") }
  client.orgs(user_name).each do |org|
    repos.concat(client.org_repos(org["login"]).map { |r| r["full_name"].split("/") })
  end
}
