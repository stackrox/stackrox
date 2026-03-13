package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
)

// ChangelogInfo contains changelog data fetched from GitHub
type ChangelogInfo struct {
	Found           bool
	Source          string   // "file" or "releases"
	URL             string   // Link to the changelog
	Content         string   // Full changelog content
	BreakingChanges []string // Extracted breaking changes
}

// GitHubClient wraps the GitHub API client
type GitHubClient struct {
	client *github.Client
	ctx    context.Context
}

// NewGitHubClient creates a new GitHub client
// If GITHUB_TOKEN is set, it will be used for authentication (higher rate limits)
func NewGitHubClient() *GitHubClient {
	client := github.NewClient(nil)

	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		client = client.WithAuthToken(token)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	_ = cancel // Will be called when client is done

	return &GitHubClient{
		client: client,
		ctx:    ctx,
	}
}

// FetchChangelog attempts to fetch changelog information for a package
func (gc *GitHubClient) FetchChangelog(basePath string) (*ChangelogInfo, error) {
	// Extract owner and repo from GitHub path
	owner, repo, ok := extractGitHubOwnerRepo(basePath)
	if !ok {
		return &ChangelogInfo{Found: false}, nil
	}

	// Try fetching CHANGELOG.md from repository
	info, err := gc.fetchChangelogFile(owner, repo)
	if err == nil && info.Found {
		return info, nil
	}

	// Fallback: try GitHub Releases
	info, err = gc.fetchReleases(owner, repo)
	if err == nil && info.Found {
		return info, nil
	}

	return &ChangelogInfo{Found: false}, nil
}

// fetchChangelogFile tries to fetch CHANGELOG.md from the repository
func (gc *GitHubClient) fetchChangelogFile(owner, repo string) (*ChangelogInfo, error) {
	// Try different common changelog filenames and branches
	filenames := []string{"CHANGELOG.md", "CHANGELOG", "CHANGES.md", "HISTORY.md"}
	branches := []string{"main", "master"}

	for _, branch := range branches {
		for _, filename := range filenames {
			url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, branch, filename)
			content, err := gc.fetchRawFile(url)
			if err == nil && content != "" {
				return &ChangelogInfo{
					Found:           true,
					Source:          "file",
					URL:             fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", owner, repo, branch, filename),
					Content:         content,
					BreakingChanges: extractBreakingChanges(content),
				}, nil
			}
		}
	}

	return &ChangelogInfo{Found: false}, nil
}

// fetchRawFile fetches a raw file from a URL
func (gc *GitHubClient) fetchRawFile(url string) (string, error) {
	req, err := http.NewRequestWithContext(gc.ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// fetchReleases fetches release notes from GitHub Releases
func (gc *GitHubClient) fetchReleases(owner, repo string) (*ChangelogInfo, error) {
	releases, _, err := gc.client.Repositories.ListReleases(gc.ctx, owner, repo, &github.ListOptions{
		PerPage: 10, // Get last 10 releases
	})
	if err != nil {
		return &ChangelogInfo{Found: false}, err
	}

	if len(releases) == 0 {
		return &ChangelogInfo{Found: false}, nil
	}

	// Combine release notes
	var sb strings.Builder
	for _, release := range releases {
		sb.WriteString(fmt.Sprintf("## %s\n\n", release.GetTagName()))
		sb.WriteString(release.GetBody())
		sb.WriteString("\n\n")
	}

	content := sb.String()
	return &ChangelogInfo{
		Found:           true,
		Source:          "releases",
		URL:             fmt.Sprintf("https://github.com/%s/%s/releases", owner, repo),
		Content:         content,
		BreakingChanges: extractBreakingChanges(content),
	}, nil
}

// extractGitHubOwnerRepo extracts owner and repo from a GitHub module path
// Returns (owner, repo, ok)
func extractGitHubOwnerRepo(path string) (string, string, bool) {
	if !strings.HasPrefix(path, "github.com/") {
		return "", "", false
	}

	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return "", "", false
	}

	return parts[1], parts[2], true
}

// extractBreakingChanges finds breaking changes in changelog content
func extractBreakingChanges(content string) []string {
	var changes []string

	// Patterns that indicate breaking changes
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?im)^[#*\s]*breaking[\s:]+(.+)$`),
		regexp.MustCompile(`(?im)^[#*\s]*⚠️(.+)$`),
		regexp.MustCompile(`(?im)^[#*\s]*removed[\s:]+(.+)$`),
		regexp.MustCompile(`(?im)\[breaking\]\s*(.+)$`),
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		for _, pattern := range patterns {
			if matches := pattern.FindStringSubmatch(line); len(matches) > 1 {
				change := strings.TrimSpace(matches[1])
				if change != "" {
					changes = append(changes, change)
				}
			}
		}
	}

	// Limit to top 5 most relevant breaking changes
	if len(changes) > 5 {
		changes = changes[:5]
	}

	return changes
}

// ExtractVersionSection extracts a specific version section from changelog
func ExtractVersionSection(content, version string) string {
	// Try to find the version header
	versionPattern := regexp.MustCompile(fmt.Sprintf(`(?mi)^#+\s*\[?%s\]?`, regexp.QuoteMeta(version)))
	lines := strings.Split(content, "\n")

	var sectionLines []string
	inSection := false
	headerLevel := 0

	for _, line := range lines {
		if versionPattern.MatchString(line) {
			inSection = true
			// Count header level (number of # characters)
			headerLevel = strings.Count(strings.SplitN(line, " ", 2)[0], "#")
			sectionLines = append(sectionLines, line)
			continue
		}

		if inSection {
			// Check if we've hit the next section header at the same or higher level
			if strings.HasPrefix(strings.TrimSpace(line), "#") {
				currentLevel := strings.Count(strings.SplitN(line, " ", 2)[0], "#")
				if currentLevel <= headerLevel {
					break
				}
			}
			sectionLines = append(sectionLines, line)
		}
	}

	if len(sectionLines) > 0 {
		result := strings.Join(sectionLines, "\n")
		// Truncate if too long
		if len(result) > 1000 {
			result = result[:1000] + "\n\n_(truncated)_"
		}
		return result
	}

	return ""
}
