package code

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// graphqlURL is a var so tests can point it at httptest.
var graphqlURL = "https://api.github.com/graphql"

// httpClient bounds every GitHub call so a stalled connection can't wedge the
// sync loop indefinitely.
var httpClient = &http.Client{Timeout: 30 * time.Second}

// isPrivate + restrictedContributionsCount detect when private commits aren't surfaced.
const contribQuery = `query($login:String!,$from:DateTime!,$to:DateTime!){
  user(login:$login){
    contributionsCollection(from:$from,to:$to){
      restrictedContributionsCount
      commitContributionsByRepository(maxRepositories:100){
        repository{ isPrivate }
        contributions(first:100){
          nodes{ occurredAt commitCount }
          pageInfo{ hasNextPage }
        }
      }
    }
  }
}`

type contribResponse struct {
	Data struct {
		User struct {
			ContributionsCollection struct {
				RestrictedContributionsCount    int `json:"restrictedContributionsCount"`
				CommitContributionsByRepository []struct {
					Repository struct {
						IsPrivate bool `json:"isPrivate"`
					} `json:"repository"`
					Contributions struct {
						Nodes []struct {
							OccurredAt  time.Time `json:"occurredAt"`
							CommitCount int       `json:"commitCount"`
						} `json:"nodes"`
						PageInfo struct {
							HasNextPage bool `json:"hasNextPage"`
						} `json:"pageInfo"`
					} `json:"contributions"`
				} `json:"commitContributionsByRepository"`
			} `json:"contributionsCollection"`
		} `json:"user"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// FetchCommitContributions returns day("2006-01-02") -> commit count for a
// single window (keep windows <= ~1 month so first:100 never truncates).
func FetchCommitContributions(ctx context.Context, log *zap.SugaredLogger, token, user string, from, to time.Time) (map[string]int, error) {
	reqBody, err := json.Marshal(map[string]any{
		"query": contribQuery,
		"variables": map[string]any{
			"login": user,
			"from":  from.UTC().Format(time.RFC3339),
			"to":    to.UTC().Format(time.RFC3339),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphqlURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("graphql request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// GitHub puts the actionable detail (bad token, RATE_LIMITED) in the body.
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("graphql status %d: %s", resp.StatusCode, bytes.TrimSpace(snippet))
	}

	var parsed contribResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if len(parsed.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s", parsed.Errors[0].Message)
	}

	cc := parsed.Data.User.ContributionsCollection
	repos := cc.CommitContributionsByRepository
	if len(repos) >= 100 {
		log.Warnw("commitContributionsByRepository hit maxRepositories cap; some repos may be omitted", "from", from, "to", to)
	}

	out := map[string]int{}
	var publicCommits, privateCommits int
	for _, repo := range repos {
		if repo.Contributions.PageInfo.HasNextPage {
			log.Warnw("commit contributions truncated for window; narrow it", "from", from, "to", to)
		}
		for _, n := range repo.Contributions.Nodes {
			out[n.OccurredAt.UTC().Format("2006-01-02")] += n.CommitCount
			if repo.Repository.IsPrivate {
				privateCommits += n.CommitCount
			} else {
				publicCommits += n.CommitCount
			}
		}
	}

	// Restricted activity but no itemized private commits => token can't read
	// private repos (or the profile setting is off); flag it. Warns, not errors:
	// restrictedContributionsCount also covers private issues/PRs.
	if cc.RestrictedContributionsCount > 0 && privateCommits == 0 {
		MetricPrivateVisible.Set(0)
		log.Warnw("private commits not surfaced: GitHub reports restricted contributions the itemized query returned none of",
			"restricted", cc.RestrictedContributionsCount, "public_commits", publicCommits, "from", from, "to", to,
			"hint", "use a classic PAT with 'repo' scope and enable Settings→Profile→'Include private contributions on my profile'")
	} else {
		MetricPrivateVisible.Set(1)
		if privateCommits > 0 {
			log.Infow("private commits captured", "private", privateCommits, "public", publicCommits, "from", from, "to", to)
		}
	}
	return out, nil
}
