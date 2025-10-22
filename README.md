# AgentBill Go SDK

OpenTelemetry-based SDK for automatically tracking and billing AI agent usage.

## Installation

### From GitHub (Recommended)
```bash
go get github.com/YOUR-ORG/agentbill-go
```

### From Source
```bash
git clone https://github.com/YOUR-ORG/agentbill-go.git
cd agentbill-go
go mod download
```

## File Structure

```
agentbill-go/
├── README.md
├── go.mod
├── agentbill.go
└── examples/
    └── basic_usage.go
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/YOUR-ORG/agentbill-go"
)

func main() {
    // Initialize AgentBill
    client := agentbill.Init(agentbill.Config{
        APIKey:     "your-api-key",
        CustomerID: "customer-123",
        Debug:      true,
    })

    // Wrap your OpenAI client
    openai := client.WrapOpenAI()

    // Use normally - all calls are automatically tracked!
    ctx := context.Background()
    response, err := openai.ChatCompletion(ctx, "gpt-4", []map[string]string{
        {"role": "user", "content": "Hello!"},
    })
    
    if err != nil {
        panic(err)
    }

    fmt.Printf("Response: %+v\n", response)

    // Flush telemetry
    client.Flush(ctx)
}
```

## Features

- ✅ Zero-config instrumentation
- ✅ Accurate token & cost tracking
- ✅ Multi-provider support (OpenAI, Anthropic)
- ✅ Rich metadata capture
- ✅ OpenTelemetry-based

## Configuration

```go
config := agentbill.Config{
    APIKey:     "your-api-key",   // Required
    BaseURL:    "https://...",     // Optional
    CustomerID: "customer-123",    // Optional
    Debug:      true,              // Optional
}

client := agentbill.Init(config)
```

## Publishing

### Prerequisites
1. Create a GitHub repository: `agentbill-go`
2. Ensure `go.mod` has correct module path

### Publishing Steps
```bash
# Tag a version
git tag v1.0.0
git push origin v1.0.0

# Go modules will automatically pick it up
# Users can install with: go get github.com/YOUR-ORG/agentbill-go@v1.0.0
```

## GitHub Repository Setup

1. Create repository: `agentbill-go`
2. Push all files including `go.mod` and `agentbill.go`
3. Tag releases following semantic versioning

## License

MIT
