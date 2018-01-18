package server

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/livegrep/livegrep/blameworthy"
	"github.com/livegrep/livegrep/server/config"
)

// Blame experiment.

type BlameData struct {
	PreviousCommit string
	NextCommit string
	Author string
	Date string
	Subject string
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

const blankHash = "                " // as wide as a displayed hash
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

	content, err := gitShowCommit(commitHash, repo.Path)
	if err != nil {
		return "", nil, errors.New("Commit does not exist")
	}
	showLines := strings.Split(content, "\n")

	obj := commitHash + ":" + path
	content, err = gitCatBlob(obj, repo.Path)
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

	if !isDiff {
		// Easy enough: simply enumerate the lines of the file.
		for i, b := range blameVector {
			f := futureVector[i]
			lines = append(lines, BlameLine{
				orBlank(b.CommitHash),
				b.LineNumber,
				orStillExists(f.CommitHash),
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
				orBlank(blameVector[j].CommitHash),
				blameVector[j].LineNumber,
				orStillExists(futureVector[k].CommitHash),
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
				orBlank(blameVector[j].CommitHash),
				blameVector[j].LineNumber,
				//"  (this commit) ",
				blankHash,
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
				//"  (this commit) ",
				blankHash,
				0,
				orStillExists(futureVector[k].CommitHash),
				futureVector[k].LineNumber,
				0,
				k + 1,
				"+",
			})
			content_lines = append(content_lines, new_lines[k])
			k++
		}
		context_to := func(til_line int) {
			distance := til_line - (j+1)
			if distance > 9 {
				for i := 0; i < 3; i++ {
					both()
				}
				for j+1 < til_line - 3 {
					j++
					k++
				}
				for i := 0; i < 3; i++ {
					lines = append(lines, BlameLine{
						"        .       ",
						0,
						"        .       ",
						//blankHash,
						0,
						0,
						0,
						"",
					})
					content_lines = append(content_lines, "")
				}
			}
			for j+1 < til_line {
				both()
			}
		}

		for _, h := range commits[i].Hunks {
			if h.OldLength > 0 {
				context_to(h.OldStart)
				// for j+1 < h.OldStart {
				// 	both()
				// }
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
		context_to(len(old_lines) + 1)
		// for j+1 < len(old_lines) + 1 {
		// 	both()
		// }
		content_lines = append(content_lines, "")
		content = strings.Join(content_lines, "\n")
	}

	// fmt.Print(lines, "\n")

	result := BlameData{
		previousCommit,
		nextCommit,
		showLines[1],
		showLines[2],
		showLines[3],
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

func gitShowCommit(commitHash string, repoPath string) (string, error) {
	// git show --pretty="%H%n%an <%ae>%n%ci%n%s" --quiet master master:travisdeps.sh
	out, err := exec.Command(
		"git", "-C", repoPath, "show", "--quiet",
		"--pretty=%H%n%an <%ae>%n%ci%n%s", commitHash,
	).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func orBlank(s string) (string) {
	if len(s) > 0 {
		return s
	}
	return blankHash
}

func orStillExists(s string) (string) {
	if len(s) > 0 {
		return s
	}
	return " (still exists) "
}

func splitLines(s string) ([]string) {
	if len(s) > 0 && s[len(s) - 1] == '\n' {
		s = s[:len(s) - 1]
	}
	return strings.Split(s, "\n")
}
