package main

import (
	"time"

	"gorm.io/gorm"
)

type Commit struct {
	gorm.Model

	Repository string
	User       string
	SHA        string
	CreatedOn  time.Time
}
