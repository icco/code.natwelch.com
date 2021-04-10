package code

import (
	"fmt"
	"time"

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
