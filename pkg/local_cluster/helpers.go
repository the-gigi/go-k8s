package local_cluster

import (
	"os/exec"
	"strings"
)

func isHomebrewInstalled() (installed bool) {
	return isToolInPath("brew")
}

func isKindInstalled() (installed bool) {
	return isToolInPath("kind")
}

func isVclusterInstalled() (installed bool) {
	return isToolInPath("vcluster")
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

// installVCluster installs vcluster via homebrew
func installVCluster() (err error) {
	err = exec.Command("brew", "install", "vcluster").Wait()
	return
}

func getVclusters() (clusters []string, err error) {
	return
}

// init checks if docker, kind and vcluster are installed
//
// If docker is not installed it panics.
// If kind is not installed it tries to install, and if that fails it panics
func init() {
	//if !isDockerInstalled() {
	//	panic("docker is not installed")
	//}
	//
	//if !isHomebrewInstalled() {
	//	panic("homebrew is not installed")
	//}
	//
	//kindInstalled := isKindInstalled()
	//vclusterInstalled := isVclusterInstalled()
	//if kindInstalled && vclusterInstalled {
	//	return
	//}
	//
	//if !kindInstalled {
	//	err := installKind()
	//	if err != nil {
	//		panic(errors.Wrap(err, "failed to install kind"))
	//	}
	//}
	//
	//if !vclusterInstalled {
	//	err := installVCluster()
	//	if err != nil {
	//		panic(errors.Wrap(err, "failed to install vcluster"))
	//	}
	//}
}
