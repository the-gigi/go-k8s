package kind_cluster

import (
	"github.com/pkg/errors"
	"os/exec"
	"strings"
)

const (
	kindVersion = "v0.12.0"
)

func isKindInstalled() (installed bool, err error) {
	bytes, err := exec.Command("which", "kind").CombinedOutput()
	if err != nil {
		return
	}
	output := string(bytes)
	installed = !strings.Contains(output, "not found")
	return
}

// installKind installed kind using Go
func installKind() (err error) {
	installed, err := isKindInstalled()
	if err != nil || installed {
		return
	}
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

func getClusters() (clusters []string, err error) {
	output, err := run("get", "clusters")
	if err != nil {
		return
	}

	for _, cluster := range strings.Split(output, "\n") {
		if cluster != "" {
			clusters = append(clusters, cluster)
		}
	}
	return
}
