package multi_cluster_lock

import (
	"os"

	"github.com/joho/godotenv"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	privateGistId = "18b035a3a81e5e64ac5c7b55301aeaf9"
)

var _ = Describe("GistClient", func() {
	var cli *GistClient

	BeforeEach(func() {
		// Try to load .env file from project root (walk up directories to find it)
		_ = godotenv.Load("../../.env")
		
		// Get GitHub token from environment variable
		token := os.Getenv("GITHUB_API_TOKEN")
		if token == "" {
			Skip("GITHUB_API_TOKEN not set in environment or .env file - skipping integration tests")
		}
		
		cli = NewGistClient(token)
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
