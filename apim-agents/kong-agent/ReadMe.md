# Guide to Integrate Kong Agent with Common Agent

This guide explains how to integrate the **Kong Agent** with the **Common Agent**.

## 1. Import the Kong Agent
In the **Common Agent** codebase, modify `internal/agent/registry.go` to import the new agent:

```go
import (
    // ... Import other agents
    kongAgent "github.com/wso2/product-apim-tooling/apim-agents/kong-agent"
)
```

## 2. Register the Kong Agent 
Update the `init()` function in `registry.go` to register the Kong Agent:

```go
func init() {
    // ... Register other agents
    agentReg.RegisterAgent("kong", &kongAgent.Agent{})
}
```

## 3. Configure the Common Agent via Helm
In the **Common Agent deployment Helm chart**, specify Kong as the gateway and define any gateway-specific configurations under the `gatewayAgent` section:

```yaml
agent:
  gateway: kong

gatewayAgent:
  key1: value1
```