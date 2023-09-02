package kind

import (
	"github.com/pkg/errors"
	"os/exec"
	"strings"
)

const (
	kindVersion = "v0.12.0"
)

func isKindInstalled() (installed bool) {
	return isToolInPath("kind")
}

func isDockerInstalled() (installed bool) {
	return isToolInPath("docker")
}

// isToolInPath checks if a tool is on the path
func isToolInPath(tool string) (inPath bool) {
	// Assume "which" will not fail ¯\_(ツ)_/¯
	bytes, _ := exec.Command("which", tool).CombinedOutput()
	inPath = !strings.Contains(string(bytes), "not found")
	return
}

// installKind installs kind via Go if it isn't installed already (you may prefer homebrew)
func installKind() (err error) {
	kindModule := "sigs.k8s.io/kind@" + kindVersion
	err = exec.Command("go", "install", kindModule).Wait()
	return
}

// run - runs a kind command
func run(args ...string) (combinedOutput string, err error) {
	if len(args) == 0 {
		err = errors.New("Run() requires at least one argument")
		return
	}

	if len(args) == 1 {
		args = strings.Split(args[0], " ")
	}

	bytes, err := exec.Command("kind", args...).CombinedOutput()
	combinedOutput = string(bytes)
	return
}

func getClusters() (clusters map[string]bool, err error) {
	clusters = make(map[string]bool)

	output, err := run("get clusters")
	if err != nil {
		return
	}

	if output == "No kind clusters found.\n" {
		return
	}

	for _, cluster := range strings.Split(output, "\n") {
		if cluster != "" {
			clusters[cluster] = true
		}
	}
	return
}

// init checks if docker and kind are installed
//
// If docker is not installed it panics.
// If kind is not installed it tries to install, and if that fails it panics
func init() {
	if !isDockerInstalled() {
		panic("docker is not installed")
	}

	if isKindInstalled() {
		return
	}

	err := installKind()
	if err != nil {
		panic(errors.Wrap(err, "failed to install kind"))
	}
}
