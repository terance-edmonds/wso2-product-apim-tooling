package events

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/wso2/product-apim-tooling/apim-agent/config"
	eventConstants "github.com/wso2/product-apim-tooling/apim-agent/pkg/eventhub/constants"
	"github.com/wso2/product-apim-tooling/apim-agent/pkg/eventhub/types"
	"github.com/wso2/product-apim-tooling/apim-agent/pkg/logging"
	msg "github.com/wso2/product-apim-tooling/apim-agent/pkg/messaging"
	internalk8sClient "github.com/wso2/product-apim-tooling/apim-agents/kong-agent/internal/k8sClient"
	logger "github.com/wso2/product-apim-tooling/apim-agents/kong-agent/internal/loggers"
	internalutils "github.com/wso2/product-apim-tooling/apim-agents/kong-agent/internal/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// var variables
var (
	ScopeList = make([]types.Scope, 0)
	// timestamps needs to be maintained as it is not guranteed to receive them in order,
	// hence older events should be discarded
	apiListTimeStampMap          = make(map[string]int64, 0)
	subsriptionsListTimeStampMap = make(map[string]int64, 0)
	applicationListTimeStampMap  = make(map[string]int64, 0)
)

// HandleLifeCycleEvents handles the events of an api through out the life cycle
func HandleLifeCycleEvents(data []byte) {
	var apiEvent msg.APIEvent
	apiLCEventErr := json.Unmarshal([]byte(string(data)), &apiEvent)
	if apiLCEventErr != nil {
		logger.LoggerMessaging.Errorf("Error occurred while unmarshalling Lifecycle event data %v", apiLCEventErr)
		return
	}
	if !belongsToTenant(apiEvent.TenantDomain) {
		logger.LoggerMessaging.Debugf("API Lifecycle event for the API %s:%s is dropped due to having non related tenantDomain : %s",
			apiEvent.APIName, apiEvent.APIVersion, apiEvent.TenantDomain)
		return
	}

	apiEventObj := types.API{UUID: apiEvent.UUID, APIID: apiEvent.APIID, Name: apiEvent.APIName,
		Context: apiEvent.APIContext, Version: apiEvent.APIVersion, Provider: apiEvent.APIProvider}

	logger.LoggerMessaging.Infof("API event data %v", apiEventObj)

	conf, _ := config.ReadConfigs()
	configuredEnvs := conf.ControlPlane.EnvironmentLabels
	logger.LoggerMessaging.Debugf("%s : %s API life cycle state change event triggered", apiEvent.APIName, apiEvent.APIVersion)
	if len(configuredEnvs) == 0 {
		configuredEnvs = append(configuredEnvs, config.DefaultGatewayName)
	}
}

// HandleAPIEvents to process api related data
func HandleAPIEvents(data []byte, eventType string, conf *config.Config, c client.Client) {
	var (
		apiEvent         msg.APIEvent
		currentTimeStamp int64 = apiEvent.Event.TimeStamp
	)

	apiEventErr := json.Unmarshal([]byte(string(data)), &apiEvent)
	if apiEventErr != nil {
		logger.LoggerMessaging.ErrorC(logging.ErrorDetails{
			Message:   fmt.Sprintf("Error occurred while unmarshalling API event data %v", apiEventErr),
			Severity:  logging.MAJOR,
			ErrorCode: 2004,
		})
		return
	}

	if !belongsToTenant(apiEvent.TenantDomain) {
		apiName := apiEvent.APIName
		if apiEvent.APIName == "" {
			apiName = apiEvent.Name
		}
		apiVersion := apiEvent.Version
		if apiEvent.Version == "" {
			apiVersion = apiEvent.Version
		}
		logger.LoggerMessaging.Debugf("API event for the API %s:%s is dropped due to having non related tenantDomain : %s",
			apiName, apiVersion, apiEvent.TenantDomain)
		return
	}

	apiEventObj := types.API{UUID: apiEvent.UUID, APIID: apiEvent.APIID, Name: apiEvent.APIName,
		Context: apiEvent.APIContext, Version: apiEvent.APIVersion, Provider: apiEvent.APIProvider}

	logger.LoggerMessaging.Infof("API event data %v", apiEventObj)

	//Per each revision, synchronization should happen.
	if strings.EqualFold(eventConstants.DeployAPIToGateway, apiEvent.Event.Type) {
		go internalutils.FetchAPIsOnEvent(conf, &apiEvent.UUID, c)
	}

	for _, env := range apiEvent.GatewayLabels {
		if isLaterEvent(apiListTimeStampMap, apiEvent.UUID+":"+env, currentTimeStamp) {
			break
		}
	}

	for _, env := range apiEvent.GatewayLabels {
		if isLaterEvent(apiListTimeStampMap, apiEvent.UUID+":"+env, currentTimeStamp) {
			break
		}
		// removeFromGateway event with multiple labels could only appear when the API is subjected
		// to delete. Hence we could simply delete after checking against just one iteration.
		if strings.EqualFold(eventConstants.RemoveAPIFromGateway, apiEvent.Event.Type) {
			internalk8sClient.UndeployAPICRs(apiEvent.UUID, c)
			break
		}
	}
}

// HandleApplicationEvents to process application related events
func HandleApplicationEvents(data []byte, eventType string) {

}

// HandleSubscriptionEvents to process subscription related events
func HandleSubscriptionEvents(data []byte, eventType string) {
}

// HandlePolicyEvents to process policy related events
func HandlePolicyEvents(data []byte, eventType string, client client.Client) {

}

// HandleAIProviderEvents to process AI Provider related events
func HandleAIProviderEvents(data []byte, eventType string, client client.Client) {

}

func belongsToTenant(tenantDomain string) bool {
	// TODO : enable this once the events are fixed in apim
	// return config.GetControlPlaneConnectedTenantDomain() == tenantDomain
	return true
}

func isLaterEvent(timeStampMap map[string]int64, mapKey string, currentTimeStamp int64) bool {
	if timeStamp, ok := timeStampMap[mapKey]; ok {
		if timeStamp > currentTimeStamp {
			return true
		}
	}
	timeStampMap[mapKey] = currentTimeStamp
	return false
}

func marshalAppAttributes(attributes interface{}) map[string]string {
	attributesMap := make(map[string]string)
	if attributes != nil {
		for key, value := range attributes.(map[string]interface{}) {
			attributesMap[key] = value.(string)
		}
	}
	return attributesMap
}
