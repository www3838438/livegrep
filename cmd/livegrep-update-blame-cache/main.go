package main

import (
	// "flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/livegrep/livegrep/blameworthy"
)

func main() {
	// flag.Parse()
	// log.SetFlags(0)
	file, err := os.Open("/home/brhodes/log2.server")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	start := time.Now()
	commits, _ := blameworthy.ParseGitLog(file, true)
	elapsed := time.Since(start)
	log.Printf("Git log loaded in %s", elapsed)

	fmt.Printf("%d commits\n", len(*commits))

	// Which file has the longest history?

	// Which file had the most lines changed?

	// Which file had the most expensive history?
	file_lengths := make(map[string]int)
	m := make(map[string]int)
	for _, commit := range *commits {
		for _, file := range commit.Files {
			for _, h := range file.Hunks {
				file_lengths[file.Path] -= h.Old_length
				file_lengths[file.Path] += h.New_length
				m[file.Path] += file_lengths[file.Path]
			}
		}
	}

	f, err := os.Create("/home/brhodes/tmp.out")
	for k, v := range m {
		fmt.Fprintf(f, "%d %s\n", v, k)
	}

	target_path := "quarantine.txt"
	target_path = "dropbox/api/v2/datatypes/team_log.py"

	var small_history blameworthy.CommitHistory
	for _, commit := range *commits {
		for _, file := range commit.Files {
			if file.Path == target_path {
				c := blameworthy.Commit{}
				c.Hash = commit.Hash
				c.Files = append(c.Files, file)
				small_history = append(small_history, c)
				break
			}
		}
	}
	// fmt.Printf("history: %v\n", small_history)
	fmt.Printf("history length: %d\n", len(small_history))

	start = time.Now()
	blameworthy.Build_index(&small_history)
	elapsed = time.Since(start)

	log.Printf("Small history loaded in %s", elapsed)

	//time.Sleep(10 * time.Second)

	//blame_index :=
	//blameworthy.Build_index(commits)
	//fmt.Print((*blame_index)["8e18c6e7:README.md"])
}
