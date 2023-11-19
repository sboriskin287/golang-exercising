package main

import (
	"flag"
	"gh/github"
	"log"
	"strconv"
)

func main() {
	var owner string
	var repo string
	flag.StringVar(&owner, "owner", "", "github owner")
	flag.StringVar(&repo, "repo", "", "github repo")
	flag.Parse()
	if owner == "" || repo == "" {
		log.Fatal("Owner and repo must be passed")
	}
	github.Owner = owner
	github.Repo = repo
	if len(flag.Args()) == 0 {
		log.Fatal("Subcommand must be passed")
	}
	switch flag.Arg(0) {
	case "get":
		if idArg := flag.Arg(1); idArg != "" {
			getSpecifiedIssue(idArg)
		} else {
			getAllIssues()
		}
	case "create":
		err := github.CreateIssue()
		if err != nil {
			log.Fatal(err)
		}
	case "edit":
		issueIdArg := flag.Arg(1)
		issueId, err := strconv.Atoi(issueIdArg)
		if err != nil {
			log.Fatalf("Invalid issue id %s", issueIdArg)
		}
		err = github.EditIssue(issueId)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func getAllIssues() {
	err := github.GetIssues()
	if err != nil {
		log.Fatal(err)
	}
}

func getSpecifiedIssue(idArg string) {
	id, err := strconv.Atoi(idArg)
	if err != nil {
		log.Fatal(err)
	}
	err = github.GetIssue(id)
	if err != nil {
		log.Fatal(err)
	}
}
