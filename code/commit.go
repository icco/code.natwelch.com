package code

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v34/github"
	"gorm.io/gorm"
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

func UserRepos(ctx context.Context, client *github.Client, user string) ([]*github.Repository, error) {
	opts := &github.RepositoryListOptions{Type: "owner", Sort: "updated", Direction: "desc"}
	repos, _, err := client.Repositories.List(ctx, user, opts)
	if err != nil {
		return nil, err
	}

	orgs, _, err := client.Organizations.List(ctx, user, nil)
	if err != nil {
		return nil, err
	}

	for _, o := range orgs {
		orepos, _, err := client.Repositories.ListByOrg(ctx, o.GetLogin(), opts)
		if err != nil {
			return nil, err
		}

		repos = append(repos, orepos...)
	}

	return repos, nil
}

func CommitsForYear(ctx context.Context, db *gorm.DB, user string, year int) (map[int]int64, error) {
	var commits []*Commit
	if err := db.Where("user = ? AND EXTRACT(YEAR FROM created_on) = ?", user, year).Order("created_on desc").Find(&commits); err != nil {
		return nil, err
	}

	stats := map[int]int64{}
	for _, c := range commits {
		_, w := c.CreatedOn.ISOWeek()
		stats[w]++
	}

	return stats, nil
}

func CommitsForAllTime(ctx context.Context, db *gorm.DB, user string) (map[string]int64, error) {
	var commits []*Commit
	if err := db.Where("user = ?", user, year).Order("created_on desc").Find(&commits); err != nil {
		return nil, err
	}

	stats := map[string]int64{}
	for _, c := range commits {
		stats[c.CreatedOn.Format("2006-01-02")]++
	}

	return stats, nil
}
