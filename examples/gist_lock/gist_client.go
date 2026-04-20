package gist_lock

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
	case http.StatusForbidden, http.StatusTooManyRequests:
		reset := parseResetAt(resp.Header, time.Now())
		gc.recordRateLimit(reset)
		return &RateLimitError{ResetAt: reset, Status: resp.StatusCode, Body: string(body)}
	default:
		return errors.Errorf("github API %s %s returned %d: %s", resp.Request.Method, resp.Request.URL.Path, resp.StatusCode, string(body))
	}
}

// get fetches the raw gist JSON.
func (gc *GistClient) get(id string) (obj map[string]any, err error) {
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
	err = json.Unmarshal(body, &obj)
	return
}

// update PATCHes the gist with the given payload.
//
// GitHub's gist PATCH does not support If-Match (the API returns 400
// "Conditional request headers are not allowed in unsafe requests"), so this
// is a last-writer-wins write. The caller detects races after the fact by
// re-reading the gist.
func (gc *GistClient) update(id string, obj map[string]any) (err error) {
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
	resp, err := gc.cli.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	return gc.checkStatus(resp, body)
}

// Get returns the content of the first file in the gist.
func (gc *GistClient) Get(id string) (data string, err error) {
	obj, err := gc.get(id)
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

// Update writes data to the first file in the gist. This is a
// last-writer-wins operation; the caller must read back to detect races.
func (gc *GistClient) Update(id, data string) (err error) {
	gist, err := gc.get(id)
	if err != nil {
		return
	}

	fileMap, _ := gist["files"].(map[string]any)
	for _, file := range fileMap {
		m, _ := file.(map[string]any)
		m["content"] = data
		break
	}
	return gc.update(id, gist)
}

func NewGistClient(accessToken string) *GistClient {
	accessToken = strings.Replace(accessToken, "\n", "", -1)
	return &GistClient{
		accessToken: accessToken,
		cli:         &http.Client{Timeout: 10 * time.Second},
	}
}
