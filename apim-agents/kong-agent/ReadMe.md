# Guide to integrate KONG Agent with APIM Agent
## For WSO2 API Manager 4.4.0

## Getting Started APIM-KONG Agent

### Setting up the development environment
    1. Install [Go 1.23](https://golang.org/dl)
    2. Fork the [repository](https://github.com/wso2/product-apim-tooling)
    3. Clone your fork into any directory.
    4. `cd` into cloned directory and then cd into `product-apim-tooling/apim-agents/kong-agent`
    5. Execute bash script in `# Usage` to build the binary.
    6. Replace the .so file in `apim-agent`
    7. Re-start the APIM-Agent deployment

# Usage

```bash
env CC=gcc go build -buildmode=plugin -o ~/Documents/wso2-pat/apim-agent/agents/kong.so ~/Documents/wso2-pat/apim-agents/kong-agent/main.go
```