package code

import (
	"context"
	"errors"
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

func (c *Commit) CheckAndSave(ctx context.Context, client *github.Client, db *gorm.DB) error {
	if c.User == "" {
		return fmt.Errorf("commit user cannot be empty")
	}

	if c.Repo == "" {
		return fmt.Errorf("commit repo cannot be empty")
	}

	if c.SHA == "" {
		return fmt.Errorf("commit SHA cannot be empty")
	}

	if result := db.WithContext(ctx).Where("user = ? AND repo = ? AND sha = ?", c.User, c.Repo, c.SHA).First(c); result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return result.Error
		}

		if result := db.WithContext(ctx).Create(c); result.Error != nil {
			return result.Error
		}
	}

	if c.CreatedOn.IsZero() {
		cmt, _, err := client.Git.GetCommit(ctx, c.User, c.Repo, c.SHA)
		if err != nil {
			return err
		}

		c.CreatedOn = cmt.GetCommitter().GetDate()
		c.User = cmt.GetCommitter().GetLogin()
	}

	result := db.WithContext(ctx).Save(c)
	return result.Error
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
		orepos, _, err := client.Repositories.ListByOrg(ctx, o.GetLogin(), nil)
		if err != nil {
			return nil, err
		}

		repos = append(repos, orepos...)
	}

	return repos, nil
}

func CommitsForYear(ctx context.Context, db *gorm.DB, user string, year int) (map[string]int64, error) {
	var commits []*Commit
	if result := db.Where("user = ? AND EXTRACT(YEAR FROM created_on) = ?", user, year).Order("created_on desc").Find(&commits); result.Error != nil {
		return nil, result.Error
	}

	stats := map[string]int64{}
	for _, c := range commits {
		stats[c.CreatedOn.Format("2006-01-02")]++
	}

	return stats, nil
}

func CommitsForAllTime(ctx context.Context, db *gorm.DB, user string) (map[string]int64, error) {
	var commits []*Commit
	if result := db.Where("user = ?", user).Order("created_on desc").Find(&commits); result.Error != nil {
		return nil, result.Error
	}

	stats := map[string]int64{}
	for _, c := range commits {
		stats[c.CreatedOn.Format("2006-01-02")]++
	}

	return stats, nil
}
