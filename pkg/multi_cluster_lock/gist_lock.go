package multi_cluster_lock

import (
	"context"
	"encoding/json"
	pkgerrors "github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"math/rand/v2"
	"time"
)

type gistLock struct {
	identity string
	gistId   string
	cli      *GistClient
}

func gistToLeaderElectionRecord(gist []byte) (record *resourcelock.LeaderElectionRecord, err error) {
	var rel resourcelock.LeaderElectionRecord
	err = json.Unmarshal(gist, &rel)
	if err != nil {
		return
	}

	record = &rel
	return
}

// Get returns the LeaderElectionRecord
func (gl *gistLock) Get(ctx context.Context) (record *resourcelock.LeaderElectionRecord, recordBytes []byte, err error) {
	defer func() {
		// Convert any error to NotFound, which is what the leader election can work with
		if err != nil {
			qualifiedResource := schema.GroupResource{
				Group:    "coordination.k8s.io",
				Resource: "Lease",
			}
			err = errors.NewNotFound(qualifiedResource, gl.gistId)
			return
		}
	}()
	gist, err := gl.cli.Get(gl.gistId)
	if err != nil {
		return
	}

	record, err = gistToLeaderElectionRecord([]byte(gist))
	if err != nil {
		return
	}

	recordBytes, err = json.Marshal(*record)
	if err != nil {
		return
	}

	// add a little random delay of up to 100 milliseconds to prevent race conditions
	delay := time.Duration(100 * rand.Float64())
	time.Sleep(delay * time.Millisecond)
	return
}

// Create attempts to create a LeaderElectionRecord
func (gl *gistLock) Create(ctx context.Context, ler resourcelock.LeaderElectionRecord) (err error) {
	return gl.Update(ctx, ler)
}

// Update will update an existing LeaderElectionRecord if not held by another actor
func (gl *gistLock) Update(ctx context.Context, ler resourcelock.LeaderElectionRecord) (err error) {
	oldLer, _, err := gl.Get(ctx)
	if err != nil {
		return
	}

	// If lock is held by another actor and has not expired yet return an error
	if oldLer.HolderIdentity != ler.HolderIdentity {
		now := time.Now()
		leaseDuration := time.Duration(oldLer.LeaseDurationSeconds) * time.Second
		validUntil := oldLer.RenewTime.Add(leaseDuration)
		leaseStillValid := validUntil.After(now)
		if leaseStillValid {
			qualifiedResource := schema.GroupResource{
				Group:    "coordination.k8s.io",
				Resource: "Lease",
			}
			err = errors.NewConflict(qualifiedResource, gl.gistId, pkgerrors.New("lease is still valid"))
			return
		}
	}

	// Update lock
	recordBytes, err := json.Marshal(ler)
	if err != nil {
		return
	}

	err = gl.cli.Update(gl.gistId, string(recordBytes))
	if err != nil {
		return
	}

	// If the lock was held by another leader ,wait for half of the renew period
	// and check if the leader has changed or not.
	// If the leader has changed then let them be the leader by returning an eror
	// However, if we are still the leader then update again to refresh the lease
	var curLer *resourcelock.LeaderElectionRecord
	if len(oldLer.HolderIdentity) == 0 || oldLer.HolderIdentity != ler.HolderIdentity {
		leaseDuration := time.Duration(ler.LeaseDurationSeconds) * time.Second
		time.Sleep(leaseDuration - time.Second)
		// Check current holder
		curLer, _, err = gl.Get(ctx)
		if err != nil {
			return
		}

		if curLer.HolderIdentity != ler.HolderIdentity {
			qualifiedResource := schema.GroupResource{
				Group:    "coordination.k8s.io",
				Resource: "Lease",
			}
			err = errors.NewConflict(qualifiedResource, gl.gistId, pkgerrors.New("there is a new leader"))
			return
		}

		// Update renew time
		ler.RenewTime = metav1.Time{time.Now()}

		// Update LER again
		ler.RenewTime = metav1.Time{time.Now()}
		err = gl.Update(ctx, ler)
	}

	return
}

// RecordEvent is used to record events. Not used by gist lock
func (gl *gistLock) RecordEvent(string) {

}

// Identity will return the locks Identity
func (gl *gistLock) Identity() string {
	return gl.identity
}

// Describe is used to convert details on current resource lock
// into a string
func (gl *gistLock) Describe() string {
	return "Github gist lock: " + gl.identity
}

func NewGistLock(identity string, gistId string, accessToken string) (lock resourcelock.Interface, err error) {
	cli := NewGistClient(accessToken)
	_, err = cli.Get(gistId)
	if err != nil {
		return
	}

	lock = &gistLock{
		identity: identity,
		gistId:   gistId,
		cli:      cli,
	}
	return
}
