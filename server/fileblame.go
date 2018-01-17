package server

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/livegrep/livegrep/blameworthy"
	"github.com/livegrep/livegrep/server/config"
)

// Blame experiment.

type BlameData struct {
	BodyClass string
	PreviousCommit string
	NextCommit string
	Lines []BlameLine
}

type BlameLine struct {
	PreviousCommit string
	PreviousLineNumber int
	NextCommit string
	NextLineNumber int
	OldLineNumber int
	NewLineNumber int
	Symbol string
}

var histories map[string]blameworthy.FileHistory

func InitBlame() (error) {
	git_stdout, err := blameworthy.RunGitLog("/home/brhodes/livegrep")
	if err != nil {
		return err
	}
	histories, err = blameworthy.ParseGitLog(git_stdout)
	if err != nil {
		return err
	}
	fmt.Printf("Loaded commits\n")
	return nil
}

func buildBlameData(
	repo config.RepoConfig,
	commitHash string,
	path string,
	isDiff bool,
) (string, *BlameData, error) {
	fmt.Print("============= ", path, "\n")
	start := time.Now()
	commits := histories[path]
	i := 0
	for ; i < len(commits); i++ {
		if commits[i].Hash == commitHash {
			break;
		}
	}
	fmt.Print(commitHash, " ", i, "\n")
	if i == len(commits) {
		return "", nil, errors.New("No blame information found")
	}

	obj := commitHash + ":" + path
	content, err := gitCatBlob(obj, repo.Path)
	if err != nil {
		return "", nil, errors.New("No such file at that commit")
	}

	blameVector, futureVector := commits.FileBlame(i)
	previousCommit := ""
	if i-1 >= 0 {
		previousCommit = commits[i-1].Hash
	}
	nextCommit := ""
	if i+1 < len(commits) {
		nextCommit = commits[i+1].Hash
	}

	lines := []BlameLine{}
	cssClass := ""

	if !isDiff {
		// Easy enough: simply enumerate the lines of the file.
		for i, b := range blameVector {
			f := futureVector[i]
			lines = append(lines, BlameLine{
				b.CommitHash,
				b.LineNumber,
				f.CommitHash,
				f.LineNumber,
				i + 1,
				0,
				"",
			})
		}
	} else {
		// More complicated: build a view of the diff by pulling
		// lines, as appropriate, from the previous or next
		// version of the file.
		new_lines := splitLines(content)

		old_lines := []string{}
		content_lines := []string{}

		if len(previousCommit) > 0 {
			obj = previousCommit + ":" + path
			content, err = gitCatBlob(obj, repo.Path)
			if err != nil {
				return "", nil, errors.New("Error getting blob")
			}
			old_lines = splitLines(content)
		}

		j := 0
		k := 0

		fmt.Print(commits[i], "\n")

		both := func() {
			lines = append(lines, BlameLine{
				blameVector[j].CommitHash,
				blameVector[j].LineNumber,
				futureVector[k].CommitHash,
				futureVector[k].LineNumber,
				j + 1,
				k + 1,
				"",
			})
			content_lines = append(content_lines, old_lines[j])
			j++
			k++

		}
		left := func() {
			lines = append(lines, BlameLine{
				blameVector[j].CommitHash,
				blameVector[j].LineNumber,
				"",
				0,
				j + 1,
				0,
				"-",
			})
			content_lines = append(content_lines, old_lines[j])
			j++
		}
		right := func() {
			lines = append(lines, BlameLine{
				"",
				0,
				futureVector[k].CommitHash,
				futureVector[k].LineNumber,
				0,
				k + 1,
				"+",
			})
			content_lines = append(content_lines, new_lines[k])
			k++
		}

		for _, h := range commits[i].Hunks {
			if h.OldLength > 0 {
				for j+1 < h.OldStart {
					both()
				}
				for m := 0; m < h.OldLength; m++ {
					left()
				}
			}
			if h.NewLength > 0 {
				for k+1 < h.NewStart {
					both()
				}
				for m := 0; m < h.NewLength; m++ {
					right()
				}
			}
		}
		for j < len(old_lines) {
			both()
		}
		content_lines = append(content_lines, "")
		content = strings.Join(content_lines, "\n")

		cssClass = "wide"
	}

	// fmt.Print(lines, "\n")

	result := BlameData{
		cssClass,
		previousCommit,
		nextCommit,
		lines,
	}

	// if !ok {
	// 	return "", nil, errors.New("No blame information found")
	// }
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

	// obj := commitHash + ":" + path
	// fmt.Print("===== ",obj, "\n")

	return content, &result, nil
}

func splitLines(s string) ([]string) {
	if len(s) > 0 && s[len(s) - 1] == '\n' {
		s = s[:len(s) - 1]
	}
	return strings.Split(s, "\n")
}
