package multi_cluster_lock

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/suite"
)

const (
	privateGistId = "18b035a3a81e5e64ac5c7b55301aeaf9"
)

type GistClientTestSuite struct {
	suite.Suite

	cli     *GistClient
	homeDir string
}

func (s *GistClientTestSuite) SetupSuite() {
	var err error
	s.homeDir, err = os.UserHomeDir()
	s.Require().Nil(err)

	filename := path.Join(s.homeDir, "github_api_token.txt")
	token, err := os.ReadFile(filename)
	s.Require().Nil(err)
	s.cli = NewGistClient(string(token))
	s.Require().Nil(err)
}

func (s *GistClientTestSuite) TestGetPrivateGist() {
	data, err := s.cli.Get(privateGistId)
	s.Require().Nil(err)
	s.Require().Equal(data, "secret")
}

func (s *GistClientTestSuite) TestUpdatePrivateGist() {
	data, err := s.cli.Get(privateGistId)
	s.Require().Nil(err)
	s.Require().Equal(data, "secret")

	err = s.cli.Update(privateGistId, "secret2")
	s.Require().Nil(err)

	data, err = s.cli.Get(privateGistId)
	s.Require().Nil(err)
	s.Require().Equal("secret2", data)

	err = s.cli.Update(privateGistId, "secret")
	s.Require().Nil(err)

	data, err = s.cli.Get(privateGistId)
	s.Require().Nil(err)
	s.Require().Equal("secret", data)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestGistClientTestSuite(t *testing.T) {
	suite.Run(t, new(GistClientTestSuite))
}
