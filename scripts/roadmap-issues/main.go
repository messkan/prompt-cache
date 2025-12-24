package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/go-github/v55/github"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

type RoadmapItem struct {
	Title     string                 `yaml:"title"`
	Body      string                 `yaml:"body"`
	Labels    []string               `yaml:"labels"`
	Assignees []string               `yaml:"assignees"`
	Meta      map[string]interface{} `yaml:"meta"`
}

func main() {
	ctx := context.Background()

	repoEnv := os.Getenv("GITHUB_REPOSITORY")
	if repoEnv == "" {
		log.Fatalf("GITHUB_REPOSITORY not set. This script expects to run inside GitHub Actions.")
	}
	parts := strings.Split(repoEnv, "/")
	if len(parts) != 2 {
		log.Fatalf("GITHUB_REPOSITORY has unexpected format: %s", repoEnv)
	}
	owner, repo := parts[0], parts[1]

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatalf("GITHUB_TOKEN not set.")
	}
	defaultAssignee := os.Getenv("DEFAULT_ASSIGNEE")

	// Read roadmap.yml from repo root
	data, err := os.ReadFile("roadmap.yml")
	if err != nil {
		log.Fatalf("failed to read roadmap.yml: %v", err)
	}

	var items []RoadmapItem
	if err := yaml.Unmarshal(data, &items); err != nil {
		log.Fatalf("failed to parse roadmap.yml: %v", err)
	}
	if len(items) == 0 {
		log.Printf("No roadmap items found in roadmap.yml. Exiting.")
		return
	}

	// GitHub client
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Fetch all existing issues (open and closed) to avoid duplicate titles
	existingTitles := make(map[string]bool)
	opt := &github.IssueListByRepoOptions{
		State:       "all",
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		issues, resp, err := client.Issues.ListByRepo(ctx, owner, repo, opt)
		if err != nil {
			log.Fatalf("failed to list issues: %v", err)
		}
		for _, is := range issues {
			if is.Title != nil {
				existingTitles[strings.TrimSpace(*is.Title)] = true
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	// Create issues for roadmap items that don't already exist
	for _, it := range items {
		title := strings.TrimSpace(it.Title)
		if title == "" {
			log.Printf("Skipping roadmap item with empty title: %+v", it)
			continue
		}
		if existingTitles[title] {
			log.Printf("Issue already exists for %q — skipping.", title)
			continue
		}

		body := it.Body
		if it.Meta != nil {
			metaBytes, _ := yaml.Marshal(it.Meta)
			body = fmt.Sprintf("%s\n\n---\n%s", body, string(metaBytes))
		}

		issueRequest := &github.IssueRequest{
			Title: &title,
			Body:  &body,
		}
		if len(it.Labels) > 0 {
			issueRequest.Labels = &it.Labels
		}

		assignees := it.Assignees
		if len(assignees) == 0 && defaultAssignee != "" {
			assignees = []string{defaultAssignee}
		}
		if len(assignees) > 0 {
			issueRequest.Assignees = &assignees
		}

		created, resp, err := client.Issues.Create(ctx, owner, repo, issueRequest)
		if err != nil {
			// If assignees caused a failure (not assignable), try without assignees
			statusCode := 0
			if resp != nil {
				statusCode = resp.StatusCode
			}
			log.Printf("Failed to create issue %q (attempt with assignees %v): %v (status %d)", title, assignees, err, statusCode)
			if len(assignees) > 0 {
				issueRequest.Assignees = nil
				created2, resp2, err2 := client.Issues.Create(ctx, owner, repo, issueRequest)
				if err2 != nil {
					statusCode2 := 0
					if resp2 != nil {
						statusCode2 = resp2.StatusCode
					}
					log.Printf("Retry without assignees also failed for %q: %v (status %d). Skipping.", title, err2, statusCode2)
					continue
				}
				log.Printf("Created issue #%d for %q (without assignees)", *created2.Number, title)
				existingTitles[title] = true
				continue
			}
			continue
		}
		log.Printf("Created issue #%d for %q", *created.Number, title)
		existingTitles[title] = true
	}
}
