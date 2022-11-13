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

# Reference

- https://github.com/kubernetes/client-go/tree/master/tools/leader-election
- https://github.com/kubernetes/client-go/tree/master/examples/leader-election
- https://itnext.io/leader-election-in-kubernetes-using-client-go-a19cbe7a9a85



