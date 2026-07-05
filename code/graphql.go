package code

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// graphqlURL is a var so tests can point it at httptest.
var graphqlURL = "https://api.github.com/graphql"

const contribQuery = `query($login:String!,$from:DateTime!,$to:DateTime!){
  user(login:$login){
    contributionsCollection(from:$from,to:$to){
      commitContributionsByRepository(maxRepositories:100){
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
				CommitContributionsByRepository []struct {
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
// single window (keep windows <= ~3 months so first:100 never truncates).
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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("graphql request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("graphql status %d", resp.StatusCode)
	}

	var parsed contribResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if len(parsed.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s", parsed.Errors[0].Message)
	}

	repos := parsed.Data.User.ContributionsCollection.CommitContributionsByRepository
	if len(repos) >= 100 {
		log.Warnw("commitContributionsByRepository hit maxRepositories cap; some repos may be omitted", "from", from, "to", to)
	}

	out := map[string]int{}
	for _, repo := range repos {
		if repo.Contributions.PageInfo.HasNextPage {
			log.Warnw("commit contributions truncated for window; narrow it", "from", from, "to", to)
		}
		for _, n := range repo.Contributions.Nodes {
			out[n.OccurredAt.UTC().Format("2006-01-02")] += n.CommitCount
		}
	}
	return out, nil
}
