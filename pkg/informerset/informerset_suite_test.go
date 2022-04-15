package informerset_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestInformerset(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Informerset Suite")
}
