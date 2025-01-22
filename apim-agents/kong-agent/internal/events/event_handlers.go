package events

import (
	"github.com/wso2/product-apim-tooling/apim-agent/config"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// HandleLifeCycleEvents handles the events of an api through out the life cycle
func HandleLifeCycleEvents(data []byte) {
}

// HandleAPIEvents to process api related data
func HandleAPIEvents(data []byte, eventType string, conf *config.Config, client client.Client) {

}

// HandleApplicationEvents to process application related events
func HandleApplicationEvents(data []byte, eventType string) {
}

// HandleSubscriptionRelatedEvents to process subscription related events
func HandleSubscriptionEvents(data []byte, eventType string) {
}

// HandlePolicyRelatedEvents to process policy related events
func HandlePolicyEvents(data []byte, eventType string, client client.Client) {

}

// HandleAIProviderEvents to process AI Provider related events
func HandleAIProviderEvents(data []byte, eventType string, client client.Client) {

}
