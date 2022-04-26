package multi_cluster_lock

import (
	"context"
	"encoding/json"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"math/rand"
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

	// add a little random delay of up to 100 milliseconds seconds to prevent race conditions
	delay := time.Duration(100 * rand.Float64())
	time.Sleep(delay * time.Millisecond)
	return
}

// Create attempts to create a LeaderElectionRecord
func (gl *gistLock) Create(ctx context.Context, ler resourcelock.LeaderElectionRecord) (err error) {
	return gl.Update(ctx, ler)
}

// Update will update and existing LeaderElectionRecord
func (gl *gistLock) Update(ctx context.Context, ler resourcelock.LeaderElectionRecord) (err error) {
	recordBytes, err := json.Marshal(ler)
	if err != nil {
		return
	}

	err = gl.cli.Update(gl.gistId, string(recordBytes))
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
	rand.Seed(time.Now().UnixNano())
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
