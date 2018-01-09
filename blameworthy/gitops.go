package blameworthy

import (
	"bufio"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const HashLength = 16		// number of hash characters to preserve

type CommitHistory []Commit;

type Commit struct {
	Hash string
	Files []FileHunks
}

type FileHunks struct {
	Path string
	Hunks []Hunk
}

type Hunk struct {
	Old_start int
	Old_length int
	New_start int
	New_length int
}

func RunGitLog(repository_path string) (io.ReadCloser, error) {
	// TODO(brhodes): remove GIT_EXTERNAL_DIFF and GIT_DIFF_OPTS
	// from the environment, just in case?
	cmd := exec.Command("/usr/bin/git",
		"-C", repository_path,
		"log",
		"-U0",
		"--format=commit %H",
		"--no-prefix",
		"--no-renames",
		"--reverse",

		// Avoid invoking custom diff commands or conversions.
		"--no-ext-diff",
		"--no-textconv",

		// Treat a merge as a simple diff against its 1st parent:
		"--first-parent",
		"-m",
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	//defer cmd.Wait()  // drat, when will we do this?
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	return stdout, nil
}

func ParseGitLog(input_stream io.ReadCloser) (*CommitHistory, error) {
	scanner := bufio.NewScanner(input_stream)

	tree := make(map[string][]string)
	tree["foo"] = append(tree["foo"], "abc123")

	var commits CommitHistory
	var commit *Commit
	var file *FileHunks

	// A dash after the second "@@" is a signal from our command
	// `strip-git-log` that it has removed the "+" and "-" lines
	// that would have followed next.
	re, _ := regexp.Compile(`@@ -(\d+),?(\d*) \+(\d+),?(\d*) @@(-?)`)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "commit ") {
			commits = append(commits, Commit{})
			commit = &commits[len(commits)-1]
			commit.Hash = line[7:7+HashLength]
		} else if strings.HasPrefix(line, "--- ") {
			commit.Files = append(commit.Files, FileHunks{})
			file = &commit.Files[len(commit.Files)-1]
			file.Path = line[4:]
			scanner.Scan()
			line2 := scanner.Text()  // the "+++" line
			if file.Path == "/dev/null" {
				file.Path = line2[4:]
			}
		} else if strings.HasPrefix(line, "@@ ") {
			result_slice := re.FindStringSubmatch(line)
			var h Hunk;
			h.Old_start, _ = strconv.Atoi(result_slice[1])
			h.Old_length = 1
			if len(result_slice[2]) > 0 {
				h.Old_length, _ = strconv.Atoi(result_slice[2])
			}
			h.New_start, _ = strconv.Atoi(result_slice[3])
			h.New_length = 1
			if len(result_slice[4]) > 0 {
				h.New_length, _ = strconv.Atoi(result_slice[4])
			}
			file.Hunks = append(file.Hunks, h)
			is_stripped := len(result_slice[5]) > 0
			if ! is_stripped {
				lines_to_skip := h.Old_length + h.New_length
				for i := 0; i < lines_to_skip; i++ {
					scanner.Scan()
				}
			}
		}
	}
	// f, err := os.Create("/home/brhodes/found-commits")
	// if err != nil {
	// 	return nil, err
	// }
	// for i := 0; i < len(commits); i++ {
	// 	fmt.Fprintf(f, "%s\n", commits[i].Hash)
	// }
	// f.Close()

	return &commits, nil
}
