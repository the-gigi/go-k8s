// Package gist_lock is a reference implementation of client-go's
// leaderelection [resourcelock.Interface] that uses a GitHub gist as the
// shared store. It is intended as an example — NOT for production use.
//
// Why not production:
//
//   - GitHub's API has primary (5000/hr) and secondary (abuse-detection) rate
//     limits. A real HA workload renewing every few seconds across multiple
//     pods can burn through both.
//   - GitHub's availability becomes part of your HA story. If api.github.com
//     is down, lease renewals fail and leadership thrashes.
//   - Gist PATCH is last-writer-wins; there is no conditional write on the
//     server, so we detect races after the fact by re-reading.
//
// For production cross-cluster leader election, implement
// [resourcelock.Interface] against a globally consistent store you already
// operate (Spanner, DynamoDB with conditional writes, etcd, Consul, and so
// on). This package is meant as the starting point for that kind of
// implementation — see [NewGistLock], [gistLock.Get], and [gistLock.Update]
// for the shape of a minimal custom lock.
//
// A working demo that uses this lock lives at
// https://github.com/the-gigi/k8s-multi-cluster-leader-election.
package gist_lock

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
	cachedAt     time.Time
}

var leaseResource = schema.GroupResource{
	Group:    "coordination.k8s.io",
	Resource: "Lease",
}

// translateError maps transport-level errors into errors the leaderelection
// package understands. NotFound becomes k8s NewNotFound. Rate limit and
// other transient errors are passed through so leaderelection logs and
// retries on the next tick, rather than treating a 403 as "the lease
// disappeared, recreate it" — which is what makes an API storm worse.
func (gl *gistLock) translateError(err error) error {
	var nf *NotFoundError
	if errors.As(err, &nf) {
		return apierrors.NewNotFound(leaseResource, gl.gistId)
	}
	return err
}

// Get returns the current LeaderElectionRecord. A short-lived local cache
// avoids a second round-trip when leaderelection calls Get() and Update()
// back-to-back in a single tick.
func (gl *gistLock) Get(ctx context.Context) (*resourcelock.LeaderElectionRecord, []byte, error) {
	if gl.cachedRecord != nil && time.Since(gl.cachedAt) < cacheTTL {
		bytes, err := json.Marshal(gl.cachedRecord)
		if err != nil {
			return nil, nil, err
		}
		return gl.cachedRecord, bytes, nil
	}

	data, err := gl.cli.Get(gl.gistId)
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
	gl.cachedAt = time.Now()
	return &record, bytes, nil
}

// Create attempts to create a LeaderElectionRecord. For a gist-backed lock
// the gist always exists, so Create reduces to an unconditional Update.
func (gl *gistLock) Create(ctx context.Context, ler resourcelock.LeaderElectionRecord) error {
	return gl.write(ctx, ler)
}

// Update writes a LeaderElectionRecord.
//
// GitHub's gist API does not support If-Match on PATCH, so this is a
// last-writer-wins write. To detect races safely, when we are taking the
// lock over from a different holder we re-read the gist after a short pause
// and return a Conflict if someone else won the race.
func (gl *gistLock) Update(ctx context.Context, ler resourcelock.LeaderElectionRecord) error {
	ler.RenewTime = metav1.NewTime(time.Now())

	priorHolder := ""
	if gl.cachedRecord != nil {
		priorHolder = gl.cachedRecord.HolderIdentity
	}

	if err := gl.write(ctx, ler); err != nil {
		return err
	}

	// Renewing our own lease is the hot path; no race to resolve.
	if priorHolder == ler.HolderIdentity {
		return nil
	}

	// Taking over from someone else: let the race settle and confirm.
	time.Sleep(200 * time.Millisecond)
	gl.invalidate()
	current, _, err := gl.Get(ctx)
	if err != nil {
		return err
	}
	if current.HolderIdentity != ler.HolderIdentity {
		return apierrors.NewConflict(leaseResource, gl.gistId, errors.New("another writer won the race"))
	}
	return nil
}

func (gl *gistLock) write(ctx context.Context, ler resourcelock.LeaderElectionRecord) error {
	recordBytes, err := json.Marshal(ler)
	if err != nil {
		return err
	}
	if err := gl.cli.Update(gl.gistId, string(recordBytes)); err != nil {
		gl.invalidate()
		return gl.translateError(err)
	}
	gl.cachedRecord = &ler
	gl.cachedAt = time.Now()
	return nil
}

func (gl *gistLock) invalidate() {
	gl.cachedRecord = nil
	gl.cachedAt = time.Time{}
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
	if _, err := cli.Get(gistId); err != nil {
		return nil, err
	}
	return &gistLock{
		identity: identity,
		gistId:   gistId,
		cli:      cli,
	}, nil
}
