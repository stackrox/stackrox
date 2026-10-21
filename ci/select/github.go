package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type githubClient struct {
	baseURL string
	token   string
	client  *http.Client
}

func newGitHubClient() githubClient {
	base := os.Getenv("GITHUB_API_URL")
	if base == "" {
		base = "https://api.github.com"
	}
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	return githubClient{baseURL: strings.TrimRight(base, "/"), token: token, client: &http.Client{Timeout: 30 * time.Second}}
}

func (g githubClient) get(ctx context.Context, path string, destination any) error {
	for attempt := range 3 {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, g.baseURL+path, nil)
		if err != nil {
			return errors.Wrap(err, "create GitHub request")
		}
		request.Header.Set("Accept", "application/vnd.github+json")
		if g.token != "" {
			request.Header.Set("Authorization", "Bearer "+g.token)
		}
		response, err := g.client.Do(request)
		if err == nil {
			if response.StatusCode == http.StatusOK {
				decodeErr := json.NewDecoder(io.LimitReader(response.Body, 32<<20)).Decode(destination)
				closeErr := response.Body.Close()
				if decodeErr != nil {
					return errors.Wrap(decodeErr, "decode GitHub response")
				}
				return errors.Wrap(closeErr, "close GitHub response")
			}
			if closeErr := response.Body.Close(); closeErr != nil {
				return errors.Wrap(closeErr, "close GitHub error response")
			}
			err = fmt.Errorf("GitHub request returned HTTP %d", response.StatusCode)
			if response.StatusCode < 500 {
				return err
			}
		}
		if attempt == 2 {
			return err
		}
		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "GitHub request canceled")
		case <-time.After(time.Duration(attempt+1) * time.Second):
		}
	}
	return errors.New("GitHub request failed")
}

func repositoryPath(repo, pr string) (string, error) {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("invalid repository %q", repo)
	}
	number, err := strconv.Atoi(pr)
	if err != nil || number < 1 {
		return "", fmt.Errorf("invalid pull request number %q", pr)
	}
	return "/repos/" + url.PathEscape(parts[0]) + "/" + url.PathEscape(parts[1]) + "/issues/" + strconv.Itoa(number), nil
}

func (g githubClient) labels(ctx context.Context, repo, pr string) ([]string, error) {
	path, err := repositoryPath(repo, pr)
	if err != nil {
		return nil, err
	}
	labels := []string{}
	for page := 1; ; page++ {
		items := []struct {
			Name string `json:"name"`
		}{}
		if err := g.get(ctx, fmt.Sprintf("%s/labels?per_page=100&page=%d", path, page), &items); err != nil {
			return nil, err
		}
		if items == nil {
			return nil, errors.New("GitHub labels response must be an array")
		}
		for _, item := range items {
			labels = append(labels, item.Name)
		}
		if len(items) < 100 {
			return labels, nil
		}
	}
}

func (g githubClient) comments(ctx context.Context, repo, pr string) ([]comment, error) {
	path, err := repositoryPath(repo, pr)
	if err != nil {
		return nil, err
	}
	comments := []comment{}
	for page := 1; ; page++ {
		items := []comment{}
		if err := g.get(ctx, fmt.Sprintf("%s/comments?per_page=100&page=%d", path, page), &items); err != nil {
			return nil, err
		}
		if items == nil {
			return nil, errors.New("GitHub comments response must be an array")
		}
		comments = append(comments, items...)
		if len(items) < 100 {
			return comments, nil
		}
	}
}
