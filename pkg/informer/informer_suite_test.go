package informer_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestInformers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Informerset Suite")
}
