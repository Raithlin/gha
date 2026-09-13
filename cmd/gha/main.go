package main

import (
	"fmt"
	"os"

	"github.com/raithlin/gha/internal/branch"
	"github.com/raithlin/gha/internal/commands"
	"github.com/raithlin/gha/internal/config"
	"github.com/raithlin/gha/internal/git"
	"github.com/raithlin/gha/internal/github"
	"github.com/raithlin/gha/internal/review"
)

func main() {
	configuration := config.Load()
	provider, err := github.NewGitHubClient(configuration.GitHubToken)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	service := review.NewService(provider)
	branchService := branch.NewService(git.NewBranchLister(""))
	resolver := git.NewRepositoryResolver(configuration.Repository)

	if err := commands.Execute(branchService, service, resolver); err != nil {
		if !commands.IsReportedError(err) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
