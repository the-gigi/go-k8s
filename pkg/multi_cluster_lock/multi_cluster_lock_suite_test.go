package multi_cluster_lock_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMultiClusterLock(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MultiClusterLock Suite")
}
