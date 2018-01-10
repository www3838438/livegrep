package server

import (
	"fmt"

	"github.com/livegrep/livegrep/blameworthy"
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
