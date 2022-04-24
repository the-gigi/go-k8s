package multi_cluster_lock

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"path"
)

const (
	privateGistId = "18b035a3a81e5e64ac5c7b55301aeaf9"
)

var _ = Describe("Gist Client Tests", Ordered, func() {
	var err error
	var cli *GistClient
	var homeDir string

	BeforeAll(func() {
		homeDir, err = os.UserHomeDir()
		Ω(err).Should(BeNil())

		filename := path.Join(homeDir, "github_api_token.txt")
		token, err := ioutil.ReadFile(filename)
		cli = NewGistClient(string(token[:len(token)-1]))
		Ω(err).Should(BeNil())
	})

	It("should get private gist successfully", func() {
		data, err := cli.Get(privateGistId)
		Ω(err).Should(BeNil())
		Ω(data).Should(Equal("secret"))
	})

	It("should update private gist successfully", func() {
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
