package blameworthy

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const hashLength = 16		// number of hash characters to preserve

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

// Given an input stream from `git log`, print out an abbreviated form
// of the log that is missing the "+" and "-" lines that give the actual
// content of each diff.  Each line like "@@ -0,0 +1,3 @@" introducing
// content will have its final double-at suffixed with a dash (like
// this: "@@-") so blameworthy will recognize that the content has been
// omitted when it reads the log as input.
func StripGitLog(input io.Reader) (error) {
	re, _ := regexp.Compile(`@@ -(\d+),?(\d*) \+(\d+),?(\d*) `)

	scanner := bufio.NewScanner(input)

	const maxCapacity = 100*1024*1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "commit ") {
			fmt.Print(line + "\n")
		} else if strings.HasPrefix(line, "--- ") {
			fmt.Print(line + "\n")
		} else if strings.HasPrefix(line, "+++ ") {
			fmt.Print(line + "\n")
		} else if strings.HasPrefix(line, "@@ ") {
			rest := line[3:]
			i := strings.Index(rest, " @@")
			fmt.Printf("@@ %s @@-\n", rest[:i])

			result_slice := re.FindStringSubmatch(line)
			//old_start, _ := strconv.Atoi(result_slice[1])
			old_length := 1
			if len(result_slice[2]) > 0 {
				old_length, _ = strconv.Atoi(result_slice[2])
			}
			//new_start, _ := strconv.Atoi(result_slice[3])
			new_length := 1
			if len(result_slice[4]) > 0 {
				new_length, _ = strconv.Atoi(result_slice[4])
			}
			lines_to_skip := old_length + new_length
			for i := 0; i < lines_to_skip; i++ {
				scanner.Scan()
			}
		}
	}
	return scanner.Err()
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
			commit.Hash = line[7:7+hashLength]
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
	return &commits, nil
}

func (commits CommitHistory) PerFile() (map[string]CommitHistory) {
	m := make(map[string]CommitHistory)
	for _, commit := range commits {
		for _, file := range commit.Files {
			key := file.Path
			history, ok := m[key]
			if !ok {
				history = CommitHistory{}
			}
			m[key] = append(history, Commit{
				commit.Hash,
				[]FileHunks{file},
			})
		}
	}
	return m

}
