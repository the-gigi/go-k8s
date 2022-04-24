package multi_cluster_lock

import (
	"context"
	"github.com/google/go-github/v43/github"
	"golang.org/x/oauth2"
)

type GistClient struct {
	client *github.Client
}

func (gg *GistClient) Get(id string) (data string, err error) {
	ctx := context.Background()
	gist, _, err := gg.client.Gists.Get(ctx, id)
	if err != nil {
		return
	}

	var file *github.GistFile
	for _, v := range gist.GetFiles() {
		file = &v
		break
	}

	data = file.GetContent()
	return
}

func (gg *GistClient) Update(id string, data string) (err error) {
	ctx := context.Background()
	gist, _, err := gg.client.Gists.Get(ctx, id)
	if err != nil {
		return
	}

	// Get the first file
	var filename github.GistFilename
	var file github.GistFile
	for k, f := range gist.GetFiles() {
		filename = k
		file = f
		break
	}

	// Update the file context
	file.Content = &data

	// Push the modified file back into the gist
	gist.Files[filename] = file

	// Write the edited gist back
	_, _, err = gg.client.Gists.Edit(ctx, id, gist)
	return
}

func NewGistClient(accessToken string) (gg *GistClient) {
	ctx := context.Background()
	sts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, sts)
	gg = &GistClient{
		client: github.NewClient(tc),
	}
	return
}
