package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

const (
	GitDir    = ".git"
	GitIgnore = ".gitignore"
	GostFile  = ".gost"
)

var (
	GOPATH = os.Getenv("GOPATH")

	dir = ""
)

func main() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		failAndUsage("Please specify a command")
	}
	cmd := strings.ToLower(os.Args[1])
	switch cmd {
	case "init":
		doinit()
	case "get":
		get()
	default:
		failAndUsage("Unknown command: %s", cmd)
	}
}

// doinit does the initialization of a gost repo
func doinit() {
	var err error
	dir, err = os.Getwd()
	if err != nil {
		log.Fatalf("Unable to determine current directory: %s", err)
	}

	if exists(GitDir) {
		log.Fatalf("%s already contains a .git folder, can't initialize gost", dir)
	}

	// Initialize a git repo
	run("git", "init")

	// Write initial files
	writeAndCommit(GitIgnore, DefaultGitIgnore)
	writeAndCommit(GostFile, DefaultGostFile)

	// Done
	log.Printf("Initialized git repo, please set your GOPATH to \"%s\" and update your PATH, e.g.", dir)
	log.Printf("  export GOPATH=\"%s\"", dir)
	log.Printf("  export PATH=\"$GOPATH/bin:$PATH\"")
	os.Setenv("GOPATH", dir)
}

// get is like go get except that it replaces github packages with subtrees,
// adds non-github packages to git as source.
func get() {
	requireGostGOPATH()

	flags := flag.NewFlagSet("get", flag.ExitOnError)
	update := flags.Bool("u", false, "update existing from remote")
	flags.Parse(os.Args[2:])
	args := flags.Args()
	if len(args) < 1 {
		log.Fatal("Please specify a package")
	}

	pkg := args[0]
	if !isGithub(pkg) {
		log.Fatal("gost only supports packages on github.com")
	}

	branch := "master"
	if len(args) > 1 {
		branch = args[1]
	}

	fetchSubtree(pkg, branch, *update, map[string]bool{})
	removeGitFolders()

	run("git", "add", "src")
	run("git", "commit", "-m", fmt.Sprintf("[gost] Added %s and its dependencies", pkg))
}

func fetchSubtree(pkg string, branch string, update bool, alreadyFetched map[string]bool) {
	// Take only the path up to the github repo
	pkgParts := strings.Split(pkg, "/")
	pkgRoot := path.Join(pkgParts[:3]...)
	if alreadyFetched[pkgRoot] {
		return
	}

	prefix := path.Join("src", pkgRoot)
	if exists(prefix) {
		if update {
			run("git", "subtree", "pull", "--squash",
				"--prefix", prefix,
				fmt.Sprintf("https://%s.git", pkgRoot),
				branch)
		} else {
			log.Printf("%s already exists, declining to add as subtree", prefix)
		}
	} else {
		run("git", "subtree", "add", "--squash",
			"--prefix", prefix,
			fmt.Sprintf("https://%s.git", pkgRoot),
			branch)
	}
	alreadyFetched[pkgRoot] = true
	fetchDeps(pkg, "master", update, alreadyFetched)
}

func fetchDeps(pkg string, branch string, update bool, alreadyFetched map[string]bool) {
	depsString := run("go", "list", "-f", "{{range .Deps}}{{.}} {{end}} {{range .TestImports}}{{.}} {{end}}", pkg)
	deps := parseDeps(depsString)

	nonGithubDeps := []string{}
	for _, dep := range deps {
		dep = strings.TrimSpace(dep)
		if dep == "" || dep == "." {
			continue
		}
		if isGithub(dep) {
			fetchSubtree(dep, branch, update, alreadyFetched)
		} else {
			nonGithubDeps = append(nonGithubDeps, dep)
		}
	}

	for _, dep := range nonGithubDeps {
		goGet(dep, update, alreadyFetched)
	}
}

func goGet(pkg string, update bool, alreadyFetched map[string]bool) {
	if alreadyFetched[pkg] {
		return
	}
	run("go", "get", pkg)
	alreadyFetched[pkg] = true
}

func writeAndCommit(file string, content string) {
	if exists(file) {
		log.Fatalf("%s already contains %s, can't initialize gost", dir, file)
	}

	err := ioutil.WriteFile(file, []byte(content), 0644)
	if err != nil {
		log.Fatalf("Unable to write %s: %s", file, err)
	}

	// Write and commit
	run("git", "add", file)
	run("git", "commit", file, "-m", "[gost] Initialized "+file)

	log.Printf("Initialized and commited %s", file)
}

func requireGostGOPATH() {
	if GOPATH == "" {
		log.Fatal("Please set your GOPATH")
	}
	requireFileInGOPATH(GostFile)
	requireFileInGOPATH(GitDir)
}

func requireFileInGOPATH(file string) {
	if !exists(path.Join(GOPATH, file)) {
		log.Fatalf("Unable to find '%s' in the GOPATH '%s', please make sure you've run gost init within your GOPATH.", file, GOPATH)
	}
}

func exists(file string) bool {
	_, err := os.Stat(file)
	return err == nil
}

func isGithub(pkg string) bool {
	return strings.Index(pkg, "github.com/") == 0
}

func parseDeps(depsString string) []string {
	depsString = strings.Replace(depsString, "[", "", -1)
	depsString = strings.Replace(depsString, "]", "", -1)
	return strings.Split(depsString, " ")
}

// removeGitFolders removes all .git folders under the src tree so that any git
// repos that didn't come from GitHub (e.g. gopkg.in) won't be treated as
// submodules.
func removeGitFolders() {
	filepath.Walk(path.Join(GOPATH, "src"), func(dir string, info os.FileInfo, oldErr error) error {
		_, file := path.Split(dir)
		if file == GitDir {
			log.Printf("Removing git folder at %s", dir)
			err := os.RemoveAll(dir)
			if err != nil {
				log.Printf("WARNING - unable to remove git folder at %s", err)
			}
		}
		return nil
	})
}

func run(prg string, args ...string) string {
	out, err := doRun(prg, args...)
	if err != nil {
		log.Fatal(err)
	}
	return out
}

func doRun(prg string, args ...string) (string, error) {
	cmd := exec.Command(prg, args...)
	log.Printf("Running %s %s", prg, strings.Join(args, " "))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s says %s", prg, string(out))
	}
	return string(out), nil
}

func failAndUsage(msg string, args ...interface{}) {
	log.Printf(msg, args...)
	log.Fatal(`
Commands:
	init - initialize a git repo in the current directory and set GOPATH to here
	get  - like go get, except that all github dependencies are imported as subtrees
`)
}

const DefaultGitIgnore = `pkg
bin
.DS_Store
*.cov
`

const DefaultGostFile = `a gost lives here`
