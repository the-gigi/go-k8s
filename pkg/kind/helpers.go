package kind

import (
	"context"
	"github.com/pkg/errors"
	"os/exec"
	"strings"
)

const (
	kindVersion = "v0.20.0" // Updated to latest stable version
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

// run - runs a kind command with context support
func run(args ...string) (combinedOutput string, err error) {
	return runWithContext(context.Background(), args...)
}

// runWithContext - runs a kind command with context support for cancellation
func runWithContext(ctx context.Context, args ...string) (combinedOutput string, err error) {
	if len(args) == 0 {
		err = errors.New("runWithContext() requires at least one argument")
		return
	}

	if len(args) == 1 {
		args = strings.Split(args[0], " ")
	}

	cmd := exec.CommandContext(ctx, "kind", args...)
	bytes, err := cmd.CombinedOutput()
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

// KindClient holds configuration for Kind operations
type KindClient struct {
	dockerInstalled bool
	kindInstalled   bool
	kindVersion     string
}

// NewKindClient creates a new Kind client with validation
func NewKindClient() (*KindClient, error) {
	client := &KindClient{}
	
	// Check Docker installation
	client.dockerInstalled = isDockerInstalled()
	if !client.dockerInstalled {
		return nil, errors.New("Docker is not installed or not accessible")
	}

	// Check Kind installation
	client.kindInstalled = isKindInstalled()
	if !client.kindInstalled {
		return nil, errors.New("Kind is not installed. Please install Kind or use InstallKind()")
	}

	// Get Kind version
	version, err := getKindVersion()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get Kind version")
	}
	client.kindVersion = version

	return client, nil
}

// InstallKind installs Kind if not already present
func InstallKind() error {
	if isKindInstalled() {
		return nil // Already installed
	}

	if err := installKind(); err != nil {
		return errors.Wrap(err, "failed to install Kind")
	}

	return nil
}

// ValidateEnvironment checks if Docker and Kind are properly installed and accessible
func ValidateEnvironment() error {
	if !isDockerInstalled() {
		return errors.New("Docker is not installed or not accessible")
	}

	if !isKindInstalled() {
		return errors.New("Kind is not installed")
	}

	// Test Docker access
	if err := testDockerAccess(); err != nil {
		return errors.Wrap(err, "Docker is not accessible")
	}

	// Test Kind access
	if err := testKindAccess(); err != nil {
		return errors.Wrap(err, "Kind is not accessible")
	}

	return nil
}

// testDockerAccess verifies Docker is accessible
func testDockerAccess() error {
	_, err := exec.Command("docker", "version").CombinedOutput()
	return err
}

// testKindAccess verifies Kind is accessible
func testKindAccess() error {
	_, err := exec.Command("kind", "version").CombinedOutput()
	return err
}

// getKindVersion returns the installed Kind version
func getKindVersion() (string, error) {
	output, err := exec.Command("kind", "version").CombinedOutput()
	if err != nil {
		return "", err
	}
	
	// Parse version from output like "kind v0.20.0 go1.20.4 linux/amd64"
	parts := strings.Fields(string(output))
	if len(parts) >= 2 {
		return parts[1], nil
	}
	
	return "", errors.New("could not parse Kind version")
}
