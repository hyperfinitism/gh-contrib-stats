package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/hyperfinitism/gh-contrib-stats/internal/config"
)

const graphqlEndpoint = "https://api.github.com/graphql"

// ContributionStats holds the fetched contribution data.
type ContributionStats struct {
	PR         int
	Commit     int
	Issue      int
	Review     int
	Discussion int
	// TopRepos sorted by weighted score descending.
	TopRepos []RepoContribution
}

type RepoContribution struct {
	Name          string
	PRs           int
	Commits       int
	Issues        int
	Reviews       int
	WeightedScore int
}

// graphQL request/response types

type gqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type gqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

const requestTimeout = 30 * time.Second

func doGraphQL(token string, req gqlRequest) (json.RawMessage, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", graphqlEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var gqlResp gqlResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	if len(gqlResp.Errors) > 0 {
		msgs := make([]string, len(gqlResp.Errors))
		for i, e := range gqlResp.Errors {
			msgs[i] = e.Message
		}
		return nil, fmt.Errorf("GraphQL errors: %s", strings.Join(msgs, "; "))
	}
	return gqlResp.Data, nil
}

const contributionsQuery = `
query($login: String!, $from: DateTime!, $to: DateTime!) {
  user(login: $login) {
    contributionsCollection(from: $from, to: $to) {
      totalCommitContributions
      totalPullRequestContributions
      totalIssueContributions
      totalPullRequestReviewContributions
      commitContributionsByRepository(maxRepositories: 100) {
        repository { nameWithOwner owner { login } }
        contributions { totalCount }
      }
      pullRequestContributionsByRepository(maxRepositories: 100) {
        repository { nameWithOwner owner { login } }
        contributions { totalCount }
      }
      issueContributionsByRepository(maxRepositories: 100) {
        repository { nameWithOwner owner { login } }
        contributions { totalCount }
      }
      pullRequestReviewContributionsByRepository(maxRepositories: 100) {
        repository { nameWithOwner owner { login } }
        contributions { totalCount }
      }
    }
  }
}
`

const searchCountQuery = `
query($q: String!) {
  search(query: $q, type: DISCUSSION, first: 1) {
    discussionCount
  }
}
`

type contributionsResponse struct {
	User struct {
		ContributionsCollection struct {
			TotalCommitContributions            int `json:"totalCommitContributions"`
			TotalPullRequestContributions       int `json:"totalPullRequestContributions"`
			TotalIssueContributions             int `json:"totalIssueContributions"`
			TotalPullRequestReviewContributions int `json:"totalPullRequestReviewContributions"`

			CommitContributionsByRepository            []repoContribEntry `json:"commitContributionsByRepository"`
			PullRequestContributionsByRepository       []repoContribEntry `json:"pullRequestContributionsByRepository"`
			IssueContributionsByRepository             []repoContribEntry `json:"issueContributionsByRepository"`
			PullRequestReviewContributionsByRepository []repoContribEntry `json:"pullRequestReviewContributionsByRepository"`
		} `json:"contributionsCollection"`
	} `json:"user"`
}

type repoContribEntry struct {
	Repository struct {
		NameWithOwner string `json:"nameWithOwner"`
		Owner         struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"repository"`
	Contributions struct {
		TotalCount int `json:"totalCount"`
	} `json:"contributions"`
}

// yearlyWindows splits a [since, until) range into at-most-one-year windows
// required by the GitHub contributionsCollection API.
func yearlyWindows(since, until time.Time) [][2]time.Time {
	var windows [][2]time.Time
	for from := since; from.Before(until); {
		to := from.AddDate(1, 0, 0)
		if to.After(until) {
			to = until
		}
		windows = append(windows, [2]time.Time{from, to})
		from = to
	}
	return windows
}

func FetchStats(cfg *config.ResolvedConfig) (*ContributionStats, error) {
	type repoAccum struct {
		prs, commits, issues, reviews int
	}
	repoMap := make(map[string]*repoAccum)
	stats := &ContributionStats{}

	addRepo := func(entries []repoContribEntry, field string) {
		for _, e := range entries {
			name := e.Repository.NameWithOwner
			owner := e.Repository.Owner.Login
			if !cfg.IncludeOwned && strings.EqualFold(owner, cfg.Username) {
				continue
			}
			if cfg.ExcludeOwners[strings.ToLower(owner)] {
				continue
			}
			if cfg.ExcludeRepos[strings.ToLower(name)] {
				continue
			}
			if repoMap[name] == nil {
				repoMap[name] = &repoAccum{}
			}
			switch field {
			case "pr":
				repoMap[name].prs += e.Contributions.TotalCount
			case "commit":
				repoMap[name].commits += e.Contributions.TotalCount
			case "issue":
				repoMap[name].issues += e.Contributions.TotalCount
			case "review":
				repoMap[name].reviews += e.Contributions.TotalCount
			}
		}
	}

	// Fetch contributions in yearly windows (GitHub API requires ≤1-year spans).
	// Note: each yearly window returns at most 100 repositories per contribution
	// type. For users contributing to >100 repos in a single year, some repos may
	// be omitted from the per-repo breakdown and top-repos list.
	for _, w := range yearlyWindows(cfg.Since, cfg.Until) {
		from, to := w[0], w[1]
		data, err := doGraphQL(cfg.Token, gqlRequest{
			Query: contributionsQuery,
			Variables: map[string]any{
				"login": cfg.Username,
				"from":  from.Format(time.RFC3339),
				"to":    to.Format(time.RFC3339),
			},
		})
		if err != nil {
			return nil, fmt.Errorf("fetching contributions (%s – %s): %w",
				from.Format("2006-01-02"), to.Format("2006-01-02"), err)
		}

		var contribs contributionsResponse
		if err := json.Unmarshal(data, &contribs); err != nil {
			return nil, fmt.Errorf("parsing contributions: %w", err)
		}

		cc := contribs.User.ContributionsCollection
		addRepo(cc.PullRequestContributionsByRepository, "pr")
		addRepo(cc.CommitContributionsByRepository, "commit")
		addRepo(cc.IssueContributionsByRepository, "issue")
		addRepo(cc.PullRequestReviewContributionsByRepository, "review")
	}

	// Sum from per-repo data so that exclude-owners/exclude-repos filters
	// are applied consistently regardless of include-owned.
	for _, r := range repoMap {
		stats.PR += r.prs
		stats.Commit += r.commits
		stats.Issue += r.issues
		stats.Review += r.reviews
	}

	// Fetch discussion count if shown or selected (either needs the data).
	if cfg.Show.Discussion || cfg.Select.Discussion {
		count, err := fetchDiscussionCount(cfg)
		if err != nil {
			return nil, err
		}
		stats.Discussion = count
	}

	// Compute top repos by weighted score.
	repos := make([]RepoContribution, 0, len(repoMap))
	for name, r := range repoMap {
		rc := RepoContribution{
			Name:    name,
			PRs:     r.prs,
			Commits: r.commits,
			Issues:  r.issues,
			Reviews: r.reviews,
		}
		rc.WeightedScore = r.prs*int(cfg.Weight.PR) +
			r.commits*int(cfg.Weight.Commit) +
			r.issues*int(cfg.Weight.Issue) +
			r.reviews*int(cfg.Weight.Review)
		repos = append(repos, rc)
	}
	sort.Slice(repos, func(i, j int) bool {
		if repos[i].WeightedScore != repos[j].WeightedScore {
			return repos[i].WeightedScore > repos[j].WeightedScore
		}
		return repos[i].Name < repos[j].Name
	})
	topN := int(cfg.Show.TopRepo)
	if topN > len(repos) {
		topN = len(repos)
	}
	stats.TopRepos = repos[:topN]

	return stats, nil
}

func fetchDiscussionCount(cfg *config.ResolvedConfig) (int, error) {
	// Build search query with filters matching the other contribution types.
	q := fmt.Sprintf("author:%s", cfg.Username)
	if !cfg.IncludeOwned {
		q += fmt.Sprintf(" -user:%s", cfg.Username)
	}
	for owner := range cfg.ExcludeOwners {
		q += fmt.Sprintf(" -org:%s -user:%s", owner, owner)
	}
	for repo := range cfg.ExcludeRepos {
		q += fmt.Sprintf(" -repo:%s", repo)
	}
	data, err := doGraphQL(cfg.Token, gqlRequest{
		Query:     searchCountQuery,
		Variables: map[string]any{"q": q},
	})
	if err != nil {
		return 0, fmt.Errorf("fetching discussion count: %w", err)
	}

	var result struct {
		Search struct {
			DiscussionCount int `json:"discussionCount"`
		} `json:"search"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return 0, fmt.Errorf("parsing discussion count: %w", err)
	}
	return result.Search.DiscussionCount, nil
}
