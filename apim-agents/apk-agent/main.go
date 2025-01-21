package main

import (
	"fmt"

	"github.com/wso2/product-apim-tooling/apim-agent/config"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type IAgent interface {
	HandleLifeCycleEvents(data []byte)
	HandleAPIEvents(data []byte, eventType string, conf *config.Config, client client.Client)
	HandleApplicationEvents(data []byte, eventType string)
	HandleSubscriptionEvents(data []byte, eventType string)
	HandlePolicyEvents(data []byte, eventType string, client client.Client)
	HandleAIProviderEvents(data []byte, eventType string, client client.Client)
}

type Agent struct{}

func (a Agent) HandleLifeCycleEvents(data []byte) {
	fmt.Println("Triggered: HandleLifeCycleEvents")
}
func (a Agent) HandleAPIEvents(data []byte, eventType string, conf *config.Config, client client.Client) {
	fmt.Println("Triggered: HandleAPIEvents")
}
func (a Agent) HandleApplicationEvents(data []byte, eventType string) {
	fmt.Println("Triggered: HandleApplicationEvents")
}
func (a Agent) HandleSubscriptionEvents(data []byte, eventType string) {
	fmt.Println("Triggered: HandleSubscriptionEvents")
}
func (a Agent) HandlePolicyEvents(data []byte, eventType string, client client.Client) {
	fmt.Println("Triggered: HandlePolicyEvents")
}
func (a Agent) HandleAIProviderEvents(data []byte, eventType string, client client.Client) {
	fmt.Println("Triggered: HandleAIProviderEvents")
}

var AgentPlugin Agent
