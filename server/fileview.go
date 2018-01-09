package server

import (
	"errors"
	"fmt"
	"net/url"
	//"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/livegrep/livegrep/blameworthy"
	"github.com/livegrep/livegrep/server/config"
)

// Mapping from known file extensions to filetype hinting.
var extToLangMap map[string]string = map[string]string{
	".AppleScript": "applescript",
	".c":           "c",
	".coffee":      "coffeescript",
	".cpp":         "cpp",
	".css":         "css",
	".go":          "go",
	".h":           "cpp",
	".html":        "xml",
	".java":        "java",
	".js":          "javascript",
	".json":        "json",
	".m":           "objectivec",
	".markdown":    "markdown",
	".md":          "markdown",
	".php":         "php",
	".pl":          "perl",
	".py":          "python",
	".rb":          "ruby",
	".rs":          "rust",
	".scala":       "scala",
	".scpt":        "applescript",
	".scss":        "scss",
	".sh":          "bash",
	".sql":         "sql",
	".swift":       "swift",
	".xml":         "xml",
	".yaml":        "yaml",
	".yml":         "yaml",
}

type breadCrumbEntry struct {
	Name string
	Path string
}

type directoryListEntry struct {
	Name          string
	Path          string
	IsDir         bool
	SymlinkTarget string
}

type fileViewerContext struct {
	PathSegments   []breadCrumbEntry
	Repo           config.RepoConfig
	Commit         string
	DirContent     *directoryContent
	FileContent    *sourceFileContent
	ExternalDomain string
}

type historyData struct {
	PreviousCommit string
	NextCommit string
	Blame []string
	Future []string
}

type sourceFileContent struct {
	Content   string
	History   historyData
	LineCount int
	Language  string
}

type directoryContent struct {
	Entries []directoryListEntry
}

type DirListingSort []directoryListEntry

func (s DirListingSort) Len() int {
	return len(s)
}

func (s DirListingSort) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s DirListingSort) Less(i, j int) bool {
	if s[i].IsDir != s[j].IsDir {
		return s[i].IsDir
	}
	return s[i].Name < s[j].Name
}

func gitRevParse(commit string, repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", commit)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func gitObjectType(obj string, repoPath string) (string, error) {
	out, err := exec.Command("git", "-C", repoPath, "cat-file", "-t", obj).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func gitCatBlob(obj string, repoPath string) (string, error) {
	out, err := exec.Command("git", "-C", repoPath, "cat-file", "blob", obj).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

type gitTreeEntry struct {
	Mode       string
	ObjectType string
	ObjectId   string
	ObjectName string
}

func gitParseTreeEntry(line string) gitTreeEntry {
	dataAndPath := strings.SplitN(line, "\t", 2)
	dataFields := strings.Split(dataAndPath[0], " ")
	return gitTreeEntry{
		Mode:       dataFields[0],
		ObjectType: dataFields[1],
		ObjectId:   dataFields[2],
		ObjectName: dataAndPath[1],
	}
}

func gitListDir(obj string, repoPath string) ([]gitTreeEntry, error) {
	out, err := exec.Command("git", "-C", repoPath, "cat-file", "-p", obj).Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	lines = lines[:len(lines)-1]
	result := make([]gitTreeEntry, len(lines))
	for i, line := range lines {
		result[i] = gitParseTreeEntry(line)
	}
	return result, nil
}

func viewUrl(repo string, path string) string {
	return "/view/" + repo + "/" + path
}

func getFileUrl(repo string, pathFromRoot string, name string, isDir bool) string {
	fileUrl := viewUrl(repo, filepath.Join(pathFromRoot, path.Clean(name)))
	if isDir {
		fileUrl += "/"
	}
	return fileUrl
}

func buildDirectoryListEntry(treeEntry gitTreeEntry, pathFromRoot string, repo config.RepoConfig) directoryListEntry {
	var fileUrl string
	var symlinkTarget string
	if treeEntry.Mode == "120000" {
		resolvedPath, err := gitCatBlob(treeEntry.ObjectId, repo.Path)
		if err == nil {
			symlinkTarget = resolvedPath
		}
	} else {
		fileUrl = getFileUrl(repo.Name, pathFromRoot, treeEntry.ObjectName, treeEntry.ObjectType == "tree")
	}
	return directoryListEntry{
		Name:          treeEntry.ObjectName,
		Path:          fileUrl,
		IsDir:         treeEntry.ObjectType == "tree",
		SymlinkTarget: symlinkTarget,
	}
}

func buildFileData(relativePath string, repo config.RepoConfig, commit string) (*fileViewerContext, error) {
	cleanPath := path.Clean(relativePath)
	if cleanPath == "." {
		cleanPath = ""
	}
	if commit == "HEAD" {
		var err error
		commit, err = gitRevParse(commit, repo.Path)
		if err != nil {
			return nil, err
		}
	}
	obj := commit + ":" + cleanPath
	pathSplits := strings.Split(cleanPath, "/")

	var fileContent *sourceFileContent
	var dirContent *directoryContent

	objectType, err := gitObjectType(obj, repo.Path)
	if err != nil {
		return nil, err
	}
	if objectType == "tree" {
		treeEntries, err := gitListDir(obj, repo.Path)
		if err != nil {
			return nil, err
		}
		dirEntries := make([]directoryListEntry, len(treeEntries))
		for i, treeEntry := range treeEntries {
			dirEntries[i] = buildDirectoryListEntry(treeEntry, cleanPath, repo)
		}
		sort.Sort(DirListingSort(dirEntries))
		dirContent = &directoryContent{
			Entries: dirEntries,
		}
	} else if objectType == "blob" {
		content, err := gitCatBlob(obj, repo.Path)
		if err != nil {
			return nil, err
		}

		fmt.Printf("============= %s\n", commit, relativePath)
		result, ok := blame_index.GetFile(commit, relativePath)
		if !ok {
			return nil, errors.New("Cannot find that commit")
		}
		b := result.Blame
		half := len(b) / 2
		prev_commit := b[0]
		next_commit := b[half]
		blame := b[1:half]
		future := b[half+1:]

		fileContent = &sourceFileContent{
			Content:   content,
			History: historyData{
				PreviousCommit: prev_commit,
				NextCommit: next_commit,
				Blame: blame,
				Future: future,
			},
			LineCount: strings.Count(string(content), "\n"),
			Language:  extToLangMap[filepath.Ext(cleanPath)],
		}
	}

	segments := make([]breadCrumbEntry, len(pathSplits))
	for i, name := range pathSplits {
		parentPath := path.Clean(strings.Join(pathSplits[0:i], "/"))
		segments[i] = breadCrumbEntry{
			Name: name,
			Path: getFileUrl(repo.Name, parentPath, name, true),
		}
	}

	externalDomain := "external viewer"
	if url, err := url.Parse(repo.Metadata["url-pattern"]); err == nil {
		externalDomain = url.Hostname()
	}

	return &fileViewerContext{
		PathSegments:   segments,
		Repo:           repo,
		Commit:         commit,
		DirContent:     dirContent,
		FileContent:    fileContent,
		ExternalDomain: externalDomain,
	}, nil
}

// Blame experiment.

var blame_index *blameworthy.BlameIndex

func Init_blame() (error) {
	git_stdout, err := blameworthy.RunGitLog("/home/brhodes/livegrep")
	if err != nil {
		return err
	}
	commits, err := blameworthy.ParseGitLog(git_stdout)
	if err != nil {
		return err
	}
	fmt.Printf("%d commits\n", len(*commits))
	blame_index = blameworthy.Build_index(commits)
	return nil
}
