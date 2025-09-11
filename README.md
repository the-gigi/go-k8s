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

