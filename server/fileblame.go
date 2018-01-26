package server

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/livegrep/livegrep/blameworthy"
	"github.com/livegrep/livegrep/server/config"
)

// Blame experiment.

type BlameData struct {
	CommitHash     string
	PreviousCommit string
	NextCommit     string
	Author         string
	Date           string
	Subject        string
	Lines          []BlameLine
	Content        string
}

type BlameLine struct {
	PreviousCommit     string
	PreviousLineNumber int
	NextCommit         string
	NextLineNumber     int
	OldLineNumber      int
	NewLineNumber      int
	Symbol             string
}

const blankHash = "                " // as wide as a displayed hash
var histories = make(map[string]*blameworthy.GitHistory)

func initBlame(cfg *config.Config) error {
	for _, r := range cfg.IndexConfig.Repositories {
		blame, ok := r.Metadata["blame"]
		if !ok {
			continue
		}
		var gitLogOutput io.ReadCloser
		if blame == "git" {
			var err error
			gitLogOutput, err = blameworthy.RunGitLog(r.Path, "HEAD")
			if err != nil {
				return err
			}
		} else {
			var err error
			gitLogOutput, err = os.Open(blame)
			if err != nil {
				return err
			}
		}
		gitHistory, err := blameworthy.ParseGitLog(gitLogOutput)
		if err != nil {
			return err
		}
		histories[r.Name] = gitHistory
	}
	return nil
}

func resolveCommit(repo config.RepoConfig, commitName string, data *BlameData) error {
	output, err := gitShowCommit(commitName, repo.Path)
	if err != nil {
		return err
	}
	lines := strings.Split(output, "\n")
	data.CommitHash = lines[0][:blameworthy.HashLength]
	data.Author = lines[1]
	data.Date = lines[2]
	return nil
}

func buildBlameData(
	repo config.RepoConfig,
	commitHash string,
	path string,
	isDiff bool,
	data *BlameData,
) error {
	start := time.Now()

	gitHistory, ok := histories[repo.Name]
	if !ok {
		return fmt.Errorf("Repo not configured for blame")
	}

	obj := commitHash + ":" + path
	content, err := gitCatBlob(obj, repo.Path)
	if err != nil {
		return err
	}

	lines := []BlameLine{}
	var result *blameworthy.BlameResult

	if !isDiff {
		// Easy enough: simply enumerate the lines of the file.

		result, err = gitHistory.FileBlame(commitHash, path)
		if err != nil {
			return err
		}

		for i, b := range result.BlameVector {
			f := result.FutureVector[i]
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

		result, err = gitHistory.DiffBlame(commitHash, path)
		if err != nil {
			return err
		}
		new_lines := splitLines(content)
		old_lines := []string{}
		if len(result.PreviousCommitHash) > 0 {
			obj := result.PreviousCommitHash + ":" + path
			content, err = gitCatBlob(obj, repo.Path)
			if err != nil {
				return fmt.Errorf("Error getting blob: %s", err)
			}
			old_lines = splitLines(content)
		}

		content_lines := []string{}
		lines, content_lines, err = buildDiff(
			result.BlameVector, result.FutureVector,
			old_lines, new_lines, result.Hunks,
			lines, content_lines)
		if err != nil {
			return err
		}

		content_lines = append(content_lines, "")
		content = strings.Join(content_lines, "\n")
	}

	elapsed := time.Since(start)
	log.Print("Whole thing took ", elapsed)

	data.PreviousCommit = result.PreviousCommitHash
	data.NextCommit = result.NextCommitHash
	data.Lines = lines
	data.Content = content
	return nil
}

func buildDiff(blameVector blameworthy.BlameVector, futureVector blameworthy.BlameVector, old_lines []string, new_lines []string, hunks []blameworthy.Hunk, lines []BlameLine, content_lines []string) ([]BlameLine, []string, error) {
	j := 0
	k := 0

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
	context := func(distance int) {
		// fmt.Print("DISTANCE ", distance, " ",
		// 	til_line, " ", j+1, "\n")
		if distance > 9 {
			for i := 0; i < 3; i++ {
				both()
				distance--
			}
			for ; distance > 3; distance-- {
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
		for ; distance > 0; distance-- {
			both()
		}
	}

	for _, h := range hunks {
		if h.OldLength > 0 {
			context(h.OldStart - (j + 1))
			for m := 0; m < h.OldLength; m++ {
				left()
			}
		}
		if h.NewLength > 0 {
			context(h.NewStart - (k + 1))
			for m := 0; m < h.NewLength; m++ {
				right()
			}
		}
	}
	end := len(old_lines) + 1
	context(end - (j + 1))

	return lines, content_lines, nil
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

func orBlank(s string) string {
	if len(s) > 0 {
		return s
	}
	return blankHash
}

func orStillExists(s string) string {
	if len(s) > 0 {
		return s
	}
	return " (still exists) "
}

func splitLines(s string) []string {
	if len(s) > 0 && s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	return strings.Split(s, "\n")
}
