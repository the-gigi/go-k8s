package multi_cluster_lock

import (
	"os"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	privateGistId = "18b035a3a81e5e64ac5c7b55301aeaf9"
)

var _ = Describe("GistClient", func() {
	var cli *GistClient
	var homeDir string

	BeforeEach(func() {
		var err error
		homeDir, err = os.UserHomeDir()
		Ω(err).Should(BeNil())

		filename := path.Join(homeDir, "github_api_token.txt")
		token, err := os.ReadFile(filename)
		Ω(err).Should(BeNil())
		cli = NewGistClient(string(token))
		Ω(cli).ShouldNot(BeNil())
	})

	It("should get private gist", func() {
		data, err := cli.Get(privateGistId)
		Ω(err).Should(BeNil())
		Ω(data).Should(Equal("secret"))
	})

	It("should update private gist", func() {
		data, err := cli.Get(privateGistId)
		Ω(err).Should(BeNil())
		Ω(data).Should(Equal("secret"))

		err = cli.Update(privateGistId, "secret2")
		Ω(err).Should(BeNil())

		data, err = cli.Get(privateGistId)
		Ω(err).Should(BeNil())
		Ω(data).Should(Equal("secret2"))

		err = cli.Update(privateGistId, "secret")
		Ω(err).Should(BeNil())

		data, err = cli.Get(privateGistId)
		Ω(err).Should(BeNil())
		Ω(data).Should(Equal("secret"))
	})
})
