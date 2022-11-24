package multi_cluster_lock

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

const (
	baseURL = "https://api.github.com/gists/"
)

type GistClient struct {
	cli         *http.Client
	accessToken string

	// client *github.Client
}

func (gc *GistClient) get(id string) (obj map[string]interface{}, err error) {
	url := baseURL + id
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}

	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("Authorization", "Bearer "+gc.accessToken)
	req.Header.Add("Accept", `application/json`)

	resp, err := gc.cli.Do(req)
	if err != nil {
		return
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &obj)
	if err != nil {
		return
	}
	return
}

func (gc *GistClient) update(id string, obj map[string]interface{}) (err error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return
	}

	url := baseURL + id
	body := bytes.NewReader(data)
	req, err := http.NewRequest("PATCH", url, body)
	if err != nil {
		return
	}

	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("Authorization", "Bearer "+gc.accessToken)
	_, err = gc.cli.Do(req)
	return
}

func (gc *GistClient) Get(id string) (data string, err error) {
	//	curl \
	//	-H "Accept: application/vnd.github+json" \
	//	-H "Authorization: Bearer <YOUR-TOKEN>" \
	//https://api.github.com/gists/GIST_ID

	obj, err := gc.get(id)
	if err != nil {
		return
	}

	fileMap := obj["files"].(map[string]interface{})
	// get content of the first file
	for _, file := range fileMap {
		rawData := file.(map[string]interface{})["content"]
		data = rawData.(string)
		break
	}

	return
}

func (gc *GistClient) Update(id string, data string) (err error) {
	//	curl \
	//	-X PATCH \
	//	-H "Accept: application/vnd.github+json" \
	//	-H "Authorization: Bearer <YOUR-TOKEN>" \
	//https://api.github.com/gists/GIST_ID \
	//	-d '{"description":"An updated gist description","files":{"README.md":{"content":"Hello World from GitHub"}}}'

	gist, err := gc.get(id)
	if err != nil {
		return
	}

	fileMap := gist["files"].(map[string]interface{})
	// get content of the first file
	for _, file := range fileMap {
		file.(map[string]interface{})["content"] = data
		break
	}

	err = gc.update(id, gist)
	return
}

func NewGistClient(accessToken string) (gc *GistClient) {
	//ctx := context.Background()
	//sts := oauth2.StaticTokenSource(
	//	&oauth2.Token{AccessToken: accessToken},
	//)
	//tc := oauth2.NewClient(ctx, sts)
	//gc = &GistClient{
	//	client: github.NewClient(tc),
	//}

	// Clean up access token from inadvertent newlines
	accessToken = strings.Replace(accessToken, "\n", "", -1)
	gc = &GistClient{
		accessToken: accessToken,
		cli:         &http.Client{},
	}
	return
}
