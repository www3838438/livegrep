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
	file, err := os.Open("/home/brhodes/log3.server")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	start := time.Now()
	histories, _ := blameworthy.ParseGitLog(file)
	elapsed := time.Since(start)
	log.Printf("Git log loaded in %s", elapsed)

	// fmt.Printf("%d commits\n", len(*commits))

	fmt.Printf("%d files\n", len(histories))

	// Which file has the longest history?

	// Which file had the most lines changed?

	// Which file had the most expensive history?
	// file_lengths := make(map[string]int)
	// m := make(map[string]int)
	// for _, commit := range *commits {
	// 	for _, file := range commit.Files {
	// 		for _, h := range file.Hunks {
	// 			file_lengths[file.Path] -= h.Old_length
	// 			file_lengths[file.Path] += h.New_length
	// 			m[file.Path] += file_lengths[file.Path]
	// 		}
	// 	}
	// }

	// f, err := os.Create("/home/brhodes/tmp.out")
	// for k, v := range m {
	// 	fmt.Fprintf(f, "%d %s\n", v, k)
	// }

	target_path := "quarantine.txt"
	target_path = "dropbox/api/v2/datatypes/team_log.py"

	small_history := histories[target_path]

	fmt.Printf("history length: %d\n", len(small_history))

	start = time.Now()
	small_history.At(len(small_history) - 2)
	elapsed = time.Since(start)

	log.Printf("Small history loaded in %s", elapsed)

	//time.Sleep(10 * time.Second)

	//blame_index :=
	//blameworthy.Build_index(commits)
	//fmt.Print((*blame_index)["8e18c6e7:README.md"])
}
