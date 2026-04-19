package multi_cluster_lock

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

// cacheTTL is how long the last-observed record is trusted without re-fetching.
// Kept short so a Get() after losing the lease still sees fresh state quickly,
// but long enough to skip a round-trip when leaderelection calls Get() and then
// Update() back-to-back within a single tick.
const cacheTTL = 500 * time.Millisecond

type gistLock struct {
	identity string
	gistId   string
	cli      *GistClient

	// cached state from the last successful Get or Update.
	// leaderelection calls Get/Update serially on one goroutine, so no mutex
	// is needed to guard these fields.
	cachedRecord *resourcelock.LeaderElectionRecord
	cachedETag   string
	cachedAt     time.Time
}

var leaseResource = schema.GroupResource{
	Group:    "coordination.k8s.io",
	Resource: "Lease",
}

// translateError maps our transport-level errors into the errors the
// leaderelection package knows how to interpret. Rate limit and other
// transient/unknown errors are passed through unchanged so leaderelection
// logs them and retries on the next tick, rather than treating a rate-limit
// as "the lease disappeared, let me recreate it" (which makes the storm
// worse).
func (gl *gistLock) translateError(err error) error {
	var nf *NotFoundError
	if errors.As(err, &nf) {
		return apierrors.NewNotFound(leaseResource, gl.gistId)
	}
	var pf *PreconditionFailedError
	if errors.As(err, &pf) {
		return apierrors.NewConflict(leaseResource, gl.gistId, err)
	}
	return err
}

// Get returns the current LeaderElectionRecord. It uses a short-lived local
// cache so back-to-back Get/Update within a single leaderelection tick does
// not cost two API calls.
func (gl *gistLock) Get(ctx context.Context) (*resourcelock.LeaderElectionRecord, []byte, error) {
	if gl.cachedRecord != nil && time.Since(gl.cachedAt) < cacheTTL {
		bytes, err := json.Marshal(gl.cachedRecord)
		if err != nil {
			return nil, nil, err
		}
		return gl.cachedRecord, bytes, nil
	}

	data, etag, err := gl.cli.Get(gl.gistId)
	if err != nil {
		return nil, nil, gl.translateError(err)
	}

	var record resourcelock.LeaderElectionRecord
	if err := json.Unmarshal([]byte(data), &record); err != nil {
		return nil, nil, err
	}

	bytes, err := json.Marshal(record)
	if err != nil {
		return nil, nil, err
	}

	gl.cachedRecord = &record
	gl.cachedETag = etag
	gl.cachedAt = time.Now()
	return &record, bytes, nil
}

// Create attempts to create a LeaderElectionRecord. For a gist-backed lock
// the gist always exists, so Create reduces to an unconditional Update.
func (gl *gistLock) Create(ctx context.Context, ler resourcelock.LeaderElectionRecord) error {
	return gl.write(ler, "")
}

// Update writes a LeaderElectionRecord using an If-Match precondition when we
// have a cached ETag. A 412 response means another writer got in first, and
// we surface that as a Conflict — leaderelection treats that as "I am not
// the leader, back off". The old implementation's read-then-sleep-then-reread
// dance is no longer needed because the server now enforces the race for us.
func (gl *gistLock) Update(ctx context.Context, ler resourcelock.LeaderElectionRecord) error {
	ler.RenewTime = metav1.NewTime(time.Now())
	return gl.write(ler, gl.cachedETag)
}

func (gl *gistLock) write(ler resourcelock.LeaderElectionRecord, ifMatch string) error {
	recordBytes, err := json.Marshal(ler)
	if err != nil {
		return err
	}

	newETag, err := gl.cli.Update(gl.gistId, string(recordBytes), ifMatch)
	if err != nil {
		// Conflict or transient error: drop the cache so the next Get() pulls
		// fresh state instead of reusing a stale ETag.
		gl.cachedRecord = nil
		gl.cachedETag = ""
		gl.cachedAt = time.Time{}
		return gl.translateError(err)
	}

	gl.cachedRecord = &ler
	gl.cachedETag = newETag
	gl.cachedAt = time.Now()
	return nil
}

func (gl *gistLock) RecordEvent(string) {}

func (gl *gistLock) Identity() string {
	return gl.identity
}

func (gl *gistLock) Describe() string {
	return "Github gist lock: " + gl.identity
}

func NewGistLock(identity, gistId, accessToken string) (resourcelock.Interface, error) {
	cli := NewGistClient(accessToken)
	if _, _, err := cli.Get(gistId); err != nil {
		return nil, err
	}
	return &gistLock{
		identity: identity,
		gistId:   gistId,
		cli:      cli,
	}, nil
}
