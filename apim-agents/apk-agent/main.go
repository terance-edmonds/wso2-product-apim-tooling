package main

import (
	"fmt"

	"github.com/wso2/product-apim-tooling/apim-agent/config"
	"github.com/wso2/product-apim-tooling/apim-apk-agent/internal/agent"
	"github.com/wso2/product-apim-tooling/apim-apk-agent/internal/events"
	"github.com/wso2/product-apim-tooling/apim-apk-agent/internal/messaging"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type Agent struct{}

func (a Agent) PreRun(conf *config.Config, scheme *runtime.Scheme) {
	agent.PreRun(conf, scheme)
}
func (a Agent) Run(conf *config.Config, mgr manager.Manager) {
	agent.Run(conf, mgr)
}
func (a Agent) ProcessEvents(conf *config.Config, client client.Client) {
	messaging.ProcessEvents(conf, client)
}
func (a Agent) HandleLifeCycleEvents(data []byte) {
	fmt.Println("Triggered: HandleLifeCycleEvents")
	events.HandleLifeCycleEvents(data)
}
func (a Agent) HandleAPIEvents(data []byte, eventType string, conf *config.Config, client client.Client) {
	fmt.Println("Triggered: HandleAPIEvents")
	events.HandleAPIEvents(data, eventType, conf, client)
}
func (a Agent) HandleApplicationEvents(data []byte, eventType string) {
	fmt.Println("Triggered: HandleApplicationEvents")
	events.HandleApplicationEvents(data, eventType)
}
func (a Agent) HandleSubscriptionEvents(data []byte, eventType string) {
	fmt.Println("Triggered: HandleSubscriptionEvents")
	events.HandleSubscriptionEvents(data, eventType)
}
func (a Agent) HandlePolicyEvents(data []byte, eventType string, client client.Client) {
	fmt.Println("Triggered: HandlePolicyEvents")
	events.HandlePolicyEvents(data, eventType, client)
}
func (a Agent) HandleAIProviderEvents(data []byte, eventType string, client client.Client) {
	fmt.Println("Triggered: HandleAIProviderEvents")
	events.HandleAIProviderEvents(data, eventType, client)
}

var AgentPlugin Agent
