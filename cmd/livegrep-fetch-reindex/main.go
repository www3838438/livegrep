package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"

	pb "github.com/livegrep/livegrep/src/proto/go_proto"
	"google.golang.org/grpc"
)

type IndexConfig struct {
	Name         string       `json:"name"`
	Repositories []RepoConfig `json:"repositories"`
}

type FileIndexConfig struct {
	Name         string       `json:"name"`
	Repositories []FileConfig `json:"fs_paths"`
}

type RepoConfig struct {
	Path      string            `json:"path"`
	Name      string            `json:"name"`
	Revisions []string          `json:"revisions"`
	Metadata  map[string]string `json:"metadata"`
}

type FileConfig struct {
	Path     string            `json:"path"`
	Name     string            `json:"name"`
	Metadata map[string]string `json:"metadata"`
}

var (
	flagCodesearch    = flag.String("codesearch", path.Join(path.Dir(os.Args[0]), "codesearch"), "Path to the `codesearch` binary")
	flagIndexPath     = flag.String("out", "livegrep.idx", "Path to write the index")
	flagCtags         = flag.Bool("build-ctags", false, "Whether to build a parallel ctags index")
	flagRevparse      = flag.Bool("revparse", true, "whether to `git rev-parse` the provided revision in generated links")
	flagSkipMissing   = flag.Bool("skip-missing", false, "skip repositories where the specified revision is missing")
	flagReloadBackend = flag.String("reload-backend", "", "Backend to send a Reload RPC to")
)

const Workers = 8

func main() {
	flag.Parse()
	log.SetFlags(0)

	if len(flag.Args()) != 1 {
		log.Fatal("Expected exactly one argument (the index json configuration)")
	}

	data, err := ioutil.ReadFile(flag.Arg(0))
	if err != nil {
		log.Fatalf(err.Error())
	}

	var cfg IndexConfig
	if err = json.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("reading %s: %s", flag.Arg(0), err.Error())
	}

	if err := checkoutRepos(&cfg.Repositories); err != nil {
		log.Fatalln(err.Error())
	}

	tmp := *flagIndexPath + ".tmp"

	args := []string{
		"--debug=ui",
		"--dump_index",
		tmp,
		"--index_only",
	}
	if *flagRevparse {
		args = append(args, "--revparse")
	}
	args = append(args, flag.Arg(0))

	cmd := exec.Command(*flagCodesearch, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalln(err)
	}

	if *flagCtags {
		// generate parallel ctags index.json
		var ctagsCfg FileIndexConfig
		ctagsCfg.Name = cfg.Name
		for _, r := range cfg.Repositories {
			workingCopyPath := strings.TrimSuffix(r.Path, ".git")
			dir, workingCopyName := filepath.Split(workingCopyPath)
			ctagsDirPath := filepath.Join(dir, "ctags", workingCopyName)
			ctagsCfg.Repositories = append(ctagsCfg.Repositories, FileConfig{
				Path:     ctagsDirPath,
				Name:     r.Name,
				Metadata: r.Metadata,
			})
		}
		tmpFile, err := ioutil.TempFile("", "")
		if err != nil {
			log.Fatalf(err.Error())
		}
		b, err := json.Marshal(ctagsCfg)
		if err != nil {
			log.Fatalf(err.Error())
		}
		if _, err := tmpFile.Write(b); err != nil {
			log.Fatalf(err.Error())
		}

		ctagsIndexPath := strings.TrimSuffix(*flagIndexPath, ".idx") + ".ctags.idx"
		tmpCtagsIndexPath := ctagsIndexPath + ".tmp"

		args := []string{
			"--debug=ui",
			"--dump_index",
			tmpCtagsIndexPath,
			"--index_only",
		}
		if *flagRevparse {
			args = append(args, "--revparse")
		}
		args = append(args, tmpFile.Name())

		log.Println("running ctags indexing! %s", strings.Join(args, " "))
		cmd := exec.Command(*flagCodesearch, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatalln(err)
		}

		os.Remove(tmpFile.Name())

		if err := os.Rename(tmpCtagsIndexPath, ctagsIndexPath); err != nil {
			log.Fatalln("ctags rename:", err.Error())
		}
	}

	if err := os.Rename(tmp, *flagIndexPath); err != nil {
		log.Fatalln("rename:", err.Error())
	}

	if *flagReloadBackend != "" {
		if err := reloadBackend(*flagReloadBackend); err != nil {
			log.Fatalln("reload:", err.Error())
		}
	}
}

func checkoutRepos(repos *[]RepoConfig) error {
	repoc := make(chan *RepoConfig)
	errc := make(chan error, Workers)
	stop := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(Workers)
	for i := 0; i < Workers; i++ {
		go func() {
			defer wg.Done()
			checkoutWorker(repoc, stop, errc)
		}()
	}

	var err error
Repos:
	for i := range *repos {
		select {
		case repoc <- &(*repos)[i]:
		case err = <-errc:
			close(stop)
			break Repos
		}
	}

	close(repoc)
	wg.Wait()
	select {
	case err = <-errc:
	default:
	}

	return err
}

func checkoutWorker(c <-chan *RepoConfig,
	stop <-chan struct{}, errc chan error) {
	for {
		select {
		case r, ok := <-c:
			if !ok {
				return
			}
			err := checkoutOne(r)
			if err != nil {
				errc <- fmt.Errorf("error processing repository %s: %s", r.Name, err.Error())
			}
		case <-stop:
			return
		}
	}
}

func retryCommand(program string, args []string) error {
	var err error
	for i := 0; i < 3; i++ {
		cmd := exec.Command("git", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err = cmd.Run(); err == nil {
			return nil
		}
	}
	return fmt.Errorf("%s %v: %s", program, args, err.Error())
}

func checkoutOne(r *RepoConfig) error {
	log.Println("Updating", r.Name)

	remote, ok := r.Metadata["remote"]
	if !ok {
		return fmt.Errorf("git remote not found in repository metadata for %s", r.Name)
	}

	out, err := exec.Command("git", "-C", r.Path, "rev-parse", "--is-bare-repository").Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return err
		}
	}
	if strings.Trim(string(out), " \n") != "true" {
		if err := os.RemoveAll(r.Path); err != nil {
			return err
		}
		if err := os.MkdirAll(r.Path, 0755); err != nil {
			return err
		}
		return retryCommand("git", []string{"clone", "--mirror", remote, r.Path})
	}

	if err := exec.Command("git", "-C", r.Path, "remote", "set-url", "origin", remote).Run(); err != nil {
		return err
	}

	if err := retryCommand("git", []string{"-C", r.Path, "fetch", "-p"}); err != nil {
		return err
	}

	if !(*flagCtags) {
		return nil
	}

	workingCopyPath := strings.TrimSuffix(r.Path, ".git")
	if err := os.MkdirAll(workingCopyPath, 0755); err != nil {
		return err
	}

	if err := exec.Command("git", "--work-tree", workingCopyPath, "--git-dir="+r.Path, "checkout", "-f", r.Revisions[0]).Run(); err != nil {
		fmt.Println("error checking out working copy")
		return err
	}
	if err := exec.Command("git", "--work-tree", workingCopyPath, "--git-dir="+r.Path, "clean", "-fdx").Run(); err != nil {
		fmt.Println("error cleaning working copy")
		return err
	}

	dir, workingCopyName := filepath.Split(workingCopyPath)
	ctagsDirPath := filepath.Join(dir, "ctags", workingCopyName)
	if err := os.MkdirAll(ctagsDirPath, 0755); err != nil {
		fmt.Println("error creating dir for ctags")
		return err
	}

	fileList, err := exec.Command("git", "-C", r.Path, "ls-files").Output()
	if err != nil {
		fmt.Println("error generating file list")
		return err
	}
	tmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		fmt.Println("error creating tmpFile")
		return err
	}
	if _, err := tmpFile.Write(fileList); err != nil {
		fmt.Println("error writing tmpFile")
		return err
	}
	if err := tmpFile.Close(); err != nil {
		fmt.Println("error closing tmpFile")
		return err
	}

	absCtagsPath, err := filepath.Abs(filepath.Join(ctagsDirPath, "ctags"))
	if err != nil {
		return err
	}
	if err := os.Remove(absCtagsPath); err != nil {
		return err
	}
	ctagsCmd := exec.Command("ctags", "--format=2", "-n", "--fields=+K", "--links=no", "-L", tmpFile.Name(), "-f", absCtagsPath)
	ctagsCmd.Dir = workingCopyPath
	if err := ctagsCmd.Run(); err != nil {
		fmt.Printf("cmd: %s\n", strings.Join(ctagsCmd.Args, " "))
		fmt.Println("error running ctags")
		return err
	}

	if err := os.Remove(tmpFile.Name()); err != nil {
		fmt.Println("error removing tmpFile")
		return err
	}

	return nil
}

func reloadBackend(addr string) error {
	client, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return err
	}

	codesearch := pb.NewCodeSearchClient(client)

	if _, err = codesearch.Reload(context.Background(), &pb.Empty{}, grpc.FailFast(false)); err != nil {
		return err
	}
	return nil
}
