# go-k8s
Various Kubernetes Go libraries, tools and services

## Usage

### Prerequisites
- Go 1.25 or later
- Docker (for running Kind clusters)
- kubectl (for interacting with Kubernetes clusters)

### Installing Test Dependencies
This project uses Ginkgo v2 and Gomega for testing. Install them with:

```bash
go install github.com/onsi/ginkgo/v2/ginkgo@latest
go install github.com/onsi/gomega@latest
```

### Building
```bash
go mod download
go build ./...
```

### Running Tests
The project uses Ginkgo v2 for testing. 

#### GitHub Token Setup (Required for multi_cluster_lock tests)
The multi_cluster_lock tests require a GitHub API token to test Gist functionality. Set up your environment:

1. Create a `.env` file in the project root:
```bash
cp .env.example .env
```

2. Add your GitHub personal access token to the `.env` file:
```
GITHUB_API_TOKEN=your_github_token_here
```

3. Generate a GitHub personal access token at: https://github.com/settings/tokens
   - Requires `gist` scope for the multi_cluster_lock tests

#### Running All Tests
Run all tests with:

```bash
go test ./...
```

Or run tests with Ginkgo for more detailed output:

```bash
ginkgo -r
```

To run tests for a specific package:

```bash
go test ./pkg/informer
# or
ginkgo ./pkg/informer
```

### Running the Example
The project includes a complete workflow example that demonstrates the Kind package functionality:

```bash
go run examples/complete_workflow.go
```

This example shows how to:
- Validate the environment
- Create and manage Kind clusters
- Work with Kubernetes contexts

## Testing Methodology

This project follows a **layered testing approach** that avoids circular dependencies and ensures reliable verification of functionality.

### Core Testing Principles

#### 1. **Independent Test Setup**
- **Tests use external tools** (kubectl) for setup and verification, not the go-k8s library itself
- **Production code** provides the abstractions being tested
- This prevents the "testing the library with itself" anti-pattern

#### 2. **Testing Architecture**
```
Test Setup/Verification:     exec.Command("kubectl", ...)  ← External tool
Code Under Test:            go-k8s library methods         ← What we're testing
```

#### 3. **Test Types**

**Unit Tests** (`*_test.go`)
- Use Go's standard testing package and Ginkgo/Gomega BDD framework
- Test individual components in isolation
- Mock external dependencies when possible
- Example: `pkg/informer/event_handler_test.go` - tests event handling without requiring a cluster

**Integration Tests** (Ginkgo suites)
- Create real Kind clusters for testing
- Use `kubectl` directly for test setup, validation, and cleanup
- Test end-to-end workflows with actual Kubernetes APIs
- Example: `pkg/kind/cluster_test.go` - creates real clusters and verifies operations

#### 4. **Test Independence**
- **Setup**: `exec.Command("kubectl", "create", "ns", "test-ns")`
- **Test**: `clientset.CoreV1().Namespaces().List()` (go-k8s code)
- **Verify**: `exec.Command("kubectl", "get", "ns")`

This ensures that if go-k8s has bugs, they don't affect test setup or verification.

#### 5. **Error Testing**
- Tests include negative cases with invalid data
- Proper error handling is verified through structured logging
- Event handler tests specifically verify correct method routing (OnAdd/OnUpdate/OnDelete)

### Running Different Test Types

**Fast unit tests only:**
```bash
go test ./pkg/informer -run TestInformerEventHandlers
```

**Integration tests (requires Docker + Kind):**
```bash
ginkgo ./pkg/kind
ginkgo ./pkg/local_cluster
```

**All tests:**
```bash
ginkgo -r
```

### Testing Dependencies
- **kubectl**: Required for test setup/verification
- **Docker**: Required for Kind cluster creation
- **Kind**: Automatically managed by the library
- **GitHub token**: Only required for multi_cluster_lock tests

This methodology ensures that:
- Tests provide independent verification of functionality
- Library bugs don't mask themselves through circular test dependencies
- Both unit and integration testing patterns are clearly defined
- The testing approach scales with the complexity of Kubernetes operations

