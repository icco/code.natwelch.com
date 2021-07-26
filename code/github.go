package code

import (
	"context"

	"github.com/google/go-github/v37/github"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

// GithubClient creates a new GithubClient.
func GithubClient(ctx context.Context, token string) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}

// RateLimited logs a warning if our error is a rate limit error.
func RateLimited(err error, log *zap.SugaredLogger) bool {
	_, ok := err.(*github.RateLimitError)
	if ok {
		log.Warnw("hit rate limit", zap.Error(err))
	}

	return ok
}
