package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"

	"github.com/google/go-github/github"
)

type IssueStub struct {
	Title string
	Body  string
	Url   string
	github.Label
}

func getGoodFirstIssues(org string) []IssueStub {
	pat := os.Getenv("GITHUB_PAT")
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: pat},
	)
	tc := oauth2.NewClient(context.Background(), ts)

	// Create a new GitHub client using the OAuth2 client
	client := github.NewClient(tc)

	// Set up the search criteria for repositories
	repoOpt := &github.RepositoryListByOrgOptions{
		Type:        "all",
		ListOptions: github.ListOptions{PerPage: 10},
	}

	// Set up the search criteria for issues
	issueOpt := &github.IssueListByRepoOptions{
		State:       "all",
		ListOptions: github.ListOptions{PerPage: 10},
	}

	// Initialize an empty slice to store the issues
	issues := []*github.Issue{}
	// Set up a loop to retrieve all repositories in the organization
	for {
		// Perform the search for repositories
		repos, resp, err := client.Repositories.ListByOrg(context.Background(), org, repoOpt)
		if err != nil {
			log.Fatal(err)
		}

		// Iterate over the repositories
		for _, repo := range repos {
			// Set up a loop to retrieve all issues for the repository
			for {
				// Perform the search for issues
				result, resp, err := client.Issues.ListByRepo(context.Background(), org, *repo.Name, issueOpt)
				if err != nil {
					log.Fatal(err)
				}

				// Add the issues to the slice
				issues = append(issues, result...)

				// Check if there are more pages of results
				if resp.NextPage == 0 {
					break
				}
				issueOpt.Page = resp.NextPage
			}
		}
		// Check if there are more pages of results
		if resp.NextPage == 0 {
			break
		}
		repoOpt.Page = resp.NextPage
	}

	// Print the number of issues
	fmt.Printf("Number of issues: %d\n", len(issues))

	targetIssues := []IssueStub{}
	// Print the issue details
	for _, issue := range issues {
		// Recent issues
		if time.Since(*issue.CreatedAt) > time.Hour*14 {
			continue
		}
		for _, label := range issue.Labels {
			if *label.Name == "good first issue" {
				newIssue := IssueStub{
					Title: issue.GetTitle(),
					Body:  issue.GetBody(),
					Url:   *issue.HTMLURL,
					Label: label,
				}
				targetIssues = append(targetIssues, newIssue)
				break
			}
		}
	}
	return targetIssues
}
func main() {
	// k8sSigs := getGoodFirstIssues("kubernetes-sigs")
	k8s := getGoodFirstIssues("kubernetes")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Set up the data for the template
		data := struct {
			Issues []IssueStub
		}{
			Issues: k8s,
		}

		// Parse the template
		tmpl, err := template.ParseFiles("template.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Execute the template
		if err := tmpl.Execute(w, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
	fmt.Println("Hosting webserver on port :8080")
	http.ListenAndServe(":8080", nil)
}
