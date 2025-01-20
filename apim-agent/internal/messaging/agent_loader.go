package messaging

import (
	"fmt"
	"plugin"

	"github.com/wso2/product-apim-tooling/apim-agent/pkg/eventhub/types"
)

// loadAgent loads the agent plugin from .so file
func loadAgent(path string) (types.Agent, error) {
	// Load the plugin
	plug, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error loading plugin: %w", err)
	}

	// Look up the `AgentPlugin` symbol
	sym, err := plug.Lookup("AgentPlugin")
	if err != nil {
		return nil, fmt.Errorf("error finding symbol: %w", err)
	}

	// Assert the symbol to the Agent interface
	agent, ok := sym.(types.Agent)
	if !ok {
		return nil, fmt.Errorf("invalid plugin type")
	}

	return agent, nil
}
