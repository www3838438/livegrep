package server

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/livegrep/livegrep/blameworthy"
	"github.com/livegrep/livegrep/server/config"
)

// Blame experiment.

var commits_by_file map[string]blameworthy.CommitHistory

func Init_blame() (error) {
	git_stdout, err := blameworthy.RunGitLog("/home/brhodes/livegrep")
	if err != nil {
		return err
	}
	commits, err := blameworthy.ParseGitLog(git_stdout)
	if err != nil {
		return err
	}
	fmt.Printf("Loaded %d commits\n", len(*commits))
	commits_by_file = commits.PerFile()
	fmt.Printf("Index inversion complete\n")
	return nil
}

func buildBlameData(
	repo config.RepoConfig,
	commitHash string,
	path string,
) (string, *blameworthy.BlameResult, error) {
	fmt.Print("============= ", path, "\n")
	start := time.Now()
	commits := commits_by_file[path]
	index := blameworthy.Build_index(commits)
	result, ok := index.GetFile(commitHash, path)
	if !ok {
		return "", nil, errors.New("No blame information found")
	}
	elapsed := time.Since(start)
	log.Print("Whole thing took ", elapsed)

	// data, err := buildFileData(path, repo, commit)
	// if err != nil {
	// 	http.Error(w, "Error reading file", 500)
	// 	return
	// }

	// script_data := &struct {
	// 	RepoInfo config.RepoConfig `json:"repo_info"`
	// 	Commit   string            `json:"commit"`
	// }{repo, commit}

	// body, err := executeTemplate(s.T.FileView, data)
	// if err != nil {
	// 	http.Error(w, err.Error(), 500)
	// 	return
	// }

	obj := commitHash + ":" + path
	fmt.Print("===== ",obj, "\n")

	content, err := gitCatBlob(obj, repo.Path)
	if err != nil {
		return "", nil, errors.New("No such file at that commit")
	}

	return content, result, nil
}
