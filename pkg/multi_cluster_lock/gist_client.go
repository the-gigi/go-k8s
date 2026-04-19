package multi_cluster_lock

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

const (
	baseURL = "https://api.github.com/gists/"
)

// RateLimitError is returned when GitHub has rate-limited the token (primary
// or secondary). ResetAt is when the limit is expected to clear.
type RateLimitError struct {
	ResetAt time.Time
	Status  int
	Body    string
}

func (e *RateLimitError) Error() string {
	wait := time.Until(e.ResetAt).Round(time.Second)
	return fmt.Sprintf("github rate limited (status %d, resets in %s)", e.Status, wait)
}

// PreconditionFailedError is returned when an If-Match PATCH loses a race
// (another writer updated the gist first).
type PreconditionFailedError struct{}

func (e *PreconditionFailedError) Error() string { return "gist precondition failed (another writer won)" }

// NotFoundError is returned when the gist does not exist.
type NotFoundError struct{ ID string }

func (e *NotFoundError) Error() string { return fmt.Sprintf("gist %q not found", e.ID) }

type GistClient struct {
	cli         *http.Client
	accessToken string

	mu               sync.Mutex
	rateLimitResetAt time.Time
}

// blockedUntilRateLimitClears returns a RateLimitError if we are still in a
// known rate-limit window, otherwise nil. Callers should skip the HTTP call
// when this returns a non-nil error.
func (gc *GistClient) blockedUntilRateLimitClears() error {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	if gc.rateLimitResetAt.IsZero() || !time.Now().Before(gc.rateLimitResetAt) {
		return nil
	}
	return &RateLimitError{ResetAt: gc.rateLimitResetAt, Status: http.StatusForbidden}
}

// recordRateLimit records the time at which GitHub says we may retry.
func (gc *GistClient) recordRateLimit(t time.Time) {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	if t.After(gc.rateLimitResetAt) {
		gc.rateLimitResetAt = t
	}
}

// parseResetAt picks the most authoritative retry time from response headers.
// Retry-After (seconds) wins over X-RateLimit-Reset (epoch seconds).
func parseResetAt(h http.Header, now time.Time) time.Time {
	if ra := h.Get("Retry-After"); ra != "" {
		if secs, err := strconv.Atoi(strings.TrimSpace(ra)); err == nil {
			return now.Add(time.Duration(secs) * time.Second)
		}
	}
	if rl := h.Get("X-RateLimit-Reset"); rl != "" {
		if epoch, err := strconv.ParseInt(strings.TrimSpace(rl), 10, 64); err == nil {
			return time.Unix(epoch, 0)
		}
	}
	return now.Add(60 * time.Second)
}

func (gc *GistClient) newRequest(method, id string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, baseURL+id, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+gc.accessToken)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	return req, nil
}

func (gc *GistClient) checkStatus(resp *http.Response, body []byte) error {
	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusNoContent:
		return nil
	case http.StatusNotFound:
		return &NotFoundError{}
	case http.StatusPreconditionFailed:
		return &PreconditionFailedError{}
	case http.StatusForbidden, http.StatusTooManyRequests:
		reset := parseResetAt(resp.Header, time.Now())
		gc.recordRateLimit(reset)
		return &RateLimitError{ResetAt: reset, Status: resp.StatusCode, Body: string(body)}
	default:
		return errors.Errorf("github API %s %s returned %d: %s", resp.Request.Method, resp.Request.URL.Path, resp.StatusCode, string(body))
	}
}

// get fetches the raw gist JSON plus the response ETag.
func (gc *GistClient) get(id string) (obj map[string]any, etag string, err error) {
	if err = gc.blockedUntilRateLimitClears(); err != nil {
		return
	}
	req, err := gc.newRequest("GET", id, nil)
	if err != nil {
		return
	}
	resp, err := gc.cli.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if err = gc.checkStatus(resp, body); err != nil {
		return
	}
	etag = resp.Header.Get("ETag")
	err = json.Unmarshal(body, &obj)
	return
}

// update PATCHes the gist with the given payload. If ifMatch is non-empty it
// is sent as the If-Match header, so the server rejects the write if the gist
// has changed since that ETag was observed.
func (gc *GistClient) update(id string, obj map[string]any, ifMatch string) (newETag string, err error) {
	if err = gc.blockedUntilRateLimitClears(); err != nil {
		return
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return
	}
	req, err := gc.newRequest("PATCH", id, bytes.NewReader(data))
	if err != nil {
		return
	}
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	resp, err := gc.cli.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if err = gc.checkStatus(resp, body); err != nil {
		return
	}
	newETag = resp.Header.Get("ETag")
	return
}

// Get returns the content of the first file in the gist plus the response ETag.
func (gc *GistClient) Get(id string) (data, etag string, err error) {
	obj, etag, err := gc.get(id)
	if err != nil {
		return
	}

	files, ok := obj["files"]
	if !ok {
		var message string
		if m, ok := obj["message"]; ok {
			message, _ = m.(string)
		}
		err = errors.Errorf("failed to get gist [%s]", message)
		return
	}

	fileMap, _ := files.(map[string]any)
	for _, file := range fileMap {
		m, _ := file.(map[string]any)
		if raw, ok := m["content"].(string); ok {
			data = raw
		}
		break
	}
	return
}

// Update writes data to the first file in the gist. If ifMatch is non-empty,
// the write only succeeds if the gist's ETag still matches.
func (gc *GistClient) Update(id, data, ifMatch string) (newETag string, err error) {
	gist, _, err := gc.get(id)
	if err != nil {
		return
	}

	fileMap, _ := gist["files"].(map[string]any)
	for _, file := range fileMap {
		m, _ := file.(map[string]any)
		m["content"] = data
		break
	}
	return gc.update(id, gist, ifMatch)
}

func NewGistClient(accessToken string) *GistClient {
	accessToken = strings.Replace(accessToken, "\n", "", -1)
	return &GistClient{
		accessToken: accessToken,
		cli:         &http.Client{Timeout: 10 * time.Second},
	}
}
