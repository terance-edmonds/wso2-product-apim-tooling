/*
 *  Copyright (c) 2024, WSO2 LLC. (http://www.wso2.org) All Rights Reserved.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 *
 */

package agent

import (
	"fmt"
	"plugin"

	"github.com/wso2/product-apim-tooling/apim-agent/internal/loggers"
	"github.com/wso2/product-apim-tooling/apim-agent/pkg/agent"
)

// loadAgent loads the agent plugin from .so file
func loadAgent(path string) (agent.Agent, error) {
	// Load the plugin
	loggers.LoggerAgent.Printf("Loading agent binary from path: %v", path)
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
	agent, ok := sym.(agent.Agent)
	if !ok {
		return nil, fmt.Errorf("invalid plugin type")
	}

	return agent, nil
}
