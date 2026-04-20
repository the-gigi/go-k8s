package gist_lock_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGistLock(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GistLock Suite")
}
