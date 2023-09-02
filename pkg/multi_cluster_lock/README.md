# Multi-cluster leader election lock

This package is designed to work with the [leader election](https://github.com/kubernetes/client-go/blob/master/tools/leaderelection) package of the client-go library.

The client-go leader election provides several lock types that work in a single Kubernetes cluster. If you want to do leader election across multiple clusters you need a different lock implementation.

# Why?

This is useful if your failure domain is an entire region. With multi-cluster leader election you can run one instance of a workload in different Kubernetes clusters across different regions. If an entire region of the leader goes down the other instances in the other clusters/regions will pick a new leader and the workload will keep running.

# How?

Provide multi-cluster lock implementations that rely on HA storage such as a cloud bucket. The multi-cluster locks implement the [resourcelock.Interface](https://github.com/kubernetes/client-go/blob/28ccde769fc5519dd84e5512ebf303ac86ef9d7c/tools/leaderelection/resourcelock/interface.go#L144) that the leaderelection package expects.

```
type Interface interface {
	// Get returns the LeaderElectionRecord
	Get(ctx context.Context) (*LeaderElectionRecord, []byte, error)

	// Create attempts to create a LeaderElectionRecord
	Create(ctx context.Context, ler LeaderElectionRecord) error

	// Update will update and existing LeaderElectionRecord
	Update(ctx context.Context, ler LeaderElectionRecord) error

	// RecordEvent is used to record events
	RecordEvent(string)

	// Identity will return the locks Identity
	Identity() string

	// Describe is used to convert details on current resource lock
	// into a string
	Describe() string
}
```

# Gist lock

The [gist_lock](gist_lock.go) is a multi-cluster lock implementation that uses a Github gist as the HA storage. It uses the [gist_client](gist_client.go) to interact with the Github gist API. The [gist_client_test](gist_client_test.go) requires Github API credentials, that it reads from a file called `github_api_token.txt` in th home directory. If you want to run the tests you need to create this file and add your Github API token. You can get an API token it here: https://github.com/settings/tokens.

Create your own private gist here:
https://gist.github.com

Change the gist ID in the `gist_client_test.go` file and you're good to go.

```
const (
	privateGistId = "18b035a3a81e5e64ac5c7b55301aeaf9"
)
```

# Reference

- https://github.com/kubernetes/client-go/tree/master/tools/leader-election
- https://github.com/kubernetes/client-go/tree/master/examples/leader-election
- https://itnext.io/leader-election-in-kubernetes-using-client-go-a19cbe7a9a85

