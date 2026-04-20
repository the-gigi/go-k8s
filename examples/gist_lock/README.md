# Gist-backed resource lock (example)

A reference implementation of client-go's [`resourcelock.Interface`](https://github.com/kubernetes/client-go/blob/master/tools/leaderelection/resourcelock/interface.go) that uses a GitHub gist as the shared store.

> This is an example. **Do not use in production.** GitHub rate limits, GitHub availability as part of your HA story, and last-writer-wins semantics on gist PATCH all make it unsuitable for real workloads. For production cross-cluster leader election, implement `resourcelock.Interface` against a globally consistent store you already operate (Spanner, DynamoDB with conditional writes, etcd, Consul, CockroachDB, etc.). This package is meant as the *starting point* for that kind of implementation.

## Why it's interesting anyway

The client-go leader election package is designed around a single interface:

```go
type Interface interface {
    Get(ctx context.Context) (*LeaderElectionRecord, []byte, error)
    Create(ctx context.Context, ler LeaderElectionRecord) error
    Update(ctx context.Context, ler LeaderElectionRecord) error
    RecordEvent(string)
    Identity() string
    Describe() string
}
```

Any HTTP-accessible key/value store with a compare-and-something write can implement it. This package is a complete worked example. Useful bits you can crib:

- `gist_lock.go` — how the interface methods map onto a single shared JSON blob, how errors translate to `apierrors.NewNotFound` / `apierrors.NewConflict`, how to cache the last observed record across back-to-back `Get`/`Update` calls.
- `gist_client.go` — GitHub-flavored rate-limit handling: typed `RateLimitError`, parsing `Retry-After` / `X-RateLimit-Reset` headers, short-circuiting requests during the rate-limit window so we don't burn more quota retrying into a closed door.

## Demo

A working end-to-end demo that spins up three virtual Kubernetes clusters and shows failover across them using this lock lives at:

👉 https://github.com/the-gigi/k8s-multi-cluster-leader-election

## Running the integration tests

The tests hit a real private gist, so they require a GitHub token with the `gist` scope.

1. Create a `.env` file in the go-k8s project root:
   ```
   GITHUB_API_TOKEN=ghp_...
   ```
   (Generate at https://github.com/settings/tokens.)
2. Create your own private gist at https://gist.github.com.
3. Update `privateGistId` in [`gist_client_test.go`](gist_client_test.go) to your gist ID.
4. From the go-k8s root: `go test ./examples/gist_lock/...`

## References

- [client-go leaderelection package](https://github.com/kubernetes/client-go/tree/master/tools/leaderelection)
- [client-go leader-election examples](https://github.com/kubernetes/client-go/tree/master/examples/leader-election)
- [Leader election in Kubernetes using client-go (Mayank Shah)](https://itnext.io/leader-election-in-kubernetes-using-client-go-a19cbe7a9a85)
- [GitHub API rate limits](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api)
