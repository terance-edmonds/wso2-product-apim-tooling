package events

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/wso2/product-apim-tooling/apim-agent/config"
	eventConstants "github.com/wso2/product-apim-tooling/apim-agent/pkg/eventhub/constants"
	"github.com/wso2/product-apim-tooling/apim-agent/pkg/eventhub/types"
	"github.com/wso2/product-apim-tooling/apim-agent/pkg/logging"
	"github.com/wso2/product-apim-tooling/apim-agent/pkg/managementserver"
	msg "github.com/wso2/product-apim-tooling/apim-agent/pkg/messaging"
	internalk8sClient "github.com/wso2/product-apim-tooling/apim-agents/kong-agent/internal/k8sClient"
	logger "github.com/wso2/product-apim-tooling/apim-agents/kong-agent/internal/loggers"
	internalutils "github.com/wso2/product-apim-tooling/apim-agents/kong-agent/internal/utils"
	pkgConstants "github.com/wso2/product-apim-tooling/apim-agents/kong-agent/pkg/constants"
	"github.com/wso2/product-apim-tooling/apim-agents/kong-agent/pkg/synchronizer"
	"github.com/wso2/product-apim-tooling/apim-agents/kong-agent/pkg/transformer"
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
func HandleApplicationEvents(data []byte, eventType string, c client.Client) {
	conf, _ := config.ReadConfigs()
	configuredEnvs := conf.ControlPlane.EnvironmentLabels
	if len(configuredEnvs) == 0 {
		configuredEnvs = append(configuredEnvs, config.DefaultGatewayName)
	}

	if strings.EqualFold(eventConstants.ApplicationRegistration, eventType) ||
		strings.EqualFold(eventConstants.RemoveApplicationKeyMapping, eventType) {
		var applicationRegistrationEvent msg.ApplicationRegistrationEvent
		appRegEventErr := json.Unmarshal([]byte(string(data)), &applicationRegistrationEvent)
		if appRegEventErr != nil {
			logger.LoggerMessaging.Errorf("Error occurred while unmarshalling Application Registration event data %v", appRegEventErr)
			return
		}

		if !belongsToTenant(applicationRegistrationEvent.TenantDomain) {
			logger.LoggerMessaging.Debugf("Application Registration event for the Consumer Key : %s is dropped due to having non related tenantDomain : %s",
				applicationRegistrationEvent.ConsumerKey, applicationRegistrationEvent.TenantDomain)
			return
		}

		logger.LoggerMessaging.Infof("============ application \n%+v\n", applicationRegistrationEvent)
		if strings.EqualFold(eventConstants.ApplicationRegistration, eventType) {
			logger.LoggerMessaging.Info("Application registration create")
			// create secret CR
			// TODO: add the real key
			rsaPublicKey := ``
			jwtCredentialSecretConfig := map[string]string{
				"algorithm":      "RS512",
				"key":            applicationRegistrationEvent.ConsumerKey,
				"rsa_public_key": rsaPublicKey,
			}
			jwtCredentialSecret := transformer.GenerateK8sCredentialSecret(applicationRegistrationEvent.ApplicationUUID, applicationRegistrationEvent.KeyType, "jwt", jwtCredentialSecretConfig)
			jwtCredentialSecret.Namespace = conf.DataPlane.Namespace
			// deploy CR
			internalk8sClient.DeploySecretCR(jwtCredentialSecret, c)
			credentials := []string{jwtCredentialSecret.ObjectMeta.Name}
			internalk8sClient.UpdateKongConsumerCredential(applicationRegistrationEvent.ApplicationUUID, c, conf, credentials)
		} else if strings.EqualFold(eventConstants.RemoveApplicationKeyMapping, eventType) {
			logger.LoggerMessaging.Info("Application registration remove")
			jwtCredentialSecretName := transformer.GenerateSecretName(applicationRegistrationEvent.ApplicationUUID, applicationRegistrationEvent.KeyType, pkgConstants.KongJwtSecretName)
			credentials := []string{jwtCredentialSecretName}
			internalk8sClient.RemoveKongConsumerCredential(applicationRegistrationEvent.ApplicationUUID, c, conf, credentials)
			internalk8sClient.UnDeploySecretCR(jwtCredentialSecretName, c, conf)
		}
	} else {
		var applicationEvent msg.ApplicationEvent
		appEventErr := json.Unmarshal([]byte(string(data)), &applicationEvent)
		if appEventErr != nil {
			logger.LoggerMessaging.Errorf("Error occurred while unmarshalling Application event data %v", appEventErr)
			return
		}

		if !belongsToTenant(applicationEvent.TenantDomain) {
			logger.LoggerMessaging.Debugf("Application event for the Application : %s (with uuid %s) is dropped due to having non related tenantDomain : %s",
				applicationEvent.ApplicationName, applicationEvent.UUID, applicationEvent.TenantDomain)
			return
		}

		logger.LoggerMessaging.Infof("Application event data %v", applicationEvent)

		if isLaterEvent(applicationListTimeStampMap, fmt.Sprint(applicationEvent.ApplicationID), applicationEvent.TimeStamp) {
			return
		}

		logger.LoggerMessaging.Infof("============ application \n%+v\n", applicationEvent)
		if applicationEvent.Event.Type == eventConstants.ApplicationCreate {
			// create consumer
			consumer := transformer.CreateConsumer(applicationEvent.UUID, applicationEvent.Subscriber)
			consumer.Namespace = conf.DataPlane.Namespace
			// deploy consumer
			internalk8sClient.DeployKongConsumerCR(consumer, c)
		} else if applicationEvent.Event.Type == eventConstants.ApplicationUpdate {
			logger.LoggerMessaging.Info("Application update")
		} else if applicationEvent.Event.Type == eventConstants.ApplicationDelete {
			internalk8sClient.UndeployAPPCRs(applicationEvent.UUID, c)
		} else {
			logger.LoggerMessaging.Warnf("Application Event Type is not recognized for the Event under "+
				"Application UUID %s", applicationEvent.UUID)
			return
		}
	}
}

// HandleSubscriptionEvents to process subscription related events
func HandleSubscriptionEvents(data []byte, eventType string, c client.Client) {
	conf, _ := config.ReadConfigs()
	configuredEnvs := conf.ControlPlane.EnvironmentLabels
	if len(configuredEnvs) == 0 {
		configuredEnvs = append(configuredEnvs, config.DefaultGatewayName)
	}

	var subscriptionEvent msg.SubscriptionEvent
	subEventErr := json.Unmarshal([]byte(string(data)), &subscriptionEvent)
	if subEventErr != nil {
		logger.LoggerMessaging.Errorf("Error occurred while unmarshalling Subscription event data %v", subEventErr)
		return
	}
	if !belongsToTenant(subscriptionEvent.TenantDomain) {
		logger.LoggerMessaging.Debugf("Subscription event for the Application : %s and API %s is dropped due to having non related tenantDomain : %s",
			subscriptionEvent.ApplicationUUID, subscriptionEvent.APIUUID, subscriptionEvent.TenantDomain)
		return
	}

	if isLaterEvent(subsriptionsListTimeStampMap, fmt.Sprint(subscriptionEvent.SubscriptionID), subscriptionEvent.TimeStamp) {
		return
	}

	logger.LoggerMessaging.Infof("===========sub \n%+v\n", subscriptionEvent)
	if subscriptionEvent.Event.Type == eventConstants.SubscriptionCreate {
		// create kong acl secret CR
		aclCredentialSecretConfig := map[string]string{
			"group": subscriptionEvent.APIUUID,
		}
		aclCredentialSecret := transformer.GenerateK8sCredentialSecret(subscriptionEvent.ApplicationUUID, "api", "acl", aclCredentialSecretConfig)
		aclCredentialSecret.Namespace = conf.DataPlane.Namespace
		// add subscription limit
		subscriptionPolicy := managementserver.GetSubscriptionPolicy(subscriptionEvent.PolicyID, subscriptionEvent.TenantDomain)
		logger.LoggerMessaging.Debugf("Subscription Policy: %v", subscriptionPolicy)
		if subscriptionPolicy.Name != "" && subscriptionPolicy.Name != "Unlimited" {
			rateLimitCRName := transformer.GeneratePolicyCRName(subscriptionPolicy.Name, subscriptionPolicy.TenantDomain, "rate-limiting", "subscription")
			annotations := []string{rateLimitCRName}
			internalk8sClient.AddKongConsumerPluginAnnotation(subscriptionEvent.ApplicationUUID, c, conf, annotations)
		}
		// update consumer credentials and deploy secret
		credentials := []string{aclCredentialSecret.ObjectMeta.Name}
		internalk8sClient.DeploySecretCR(aclCredentialSecret, c)
		internalk8sClient.UpdateKongConsumerCredential(subscriptionEvent.ApplicationUUID, c, conf, credentials)
	} else if subscriptionEvent.Event.Type == eventConstants.SubscriptionUpdate {
		logger.LoggerMessaging.Info("Update Sub")
	} else if subscriptionEvent.Event.Type == eventConstants.SubscriptionDelete {
		// remove subscription from consumer
		aclCredentialSecretName := transformer.GenerateSecretName(subscriptionEvent.ApplicationUUID, "api", "acl")
		credentials := []string{aclCredentialSecretName}
		internalk8sClient.RemoveKongConsumerCredential(subscriptionEvent.ApplicationUUID, c, conf, credentials)
		// remove subscription limit from consumer
		subscriptionPolicy := managementserver.GetSubscriptionPolicy(subscriptionEvent.PolicyID, subscriptionEvent.TenantDomain)
		if subscriptionPolicy.Name != "" && subscriptionPolicy.Name != "Unlimited" {
			rateLimitCRName := transformer.GeneratePolicyCRName(subscriptionPolicy.Name, subscriptionPolicy.TenantDomain, "rate-limiting", "subscription")
			annotations := []string{rateLimitCRName}
			internalk8sClient.RemoveKongConsumerPluginAnnotation(subscriptionEvent.ApplicationUUID, c, conf, annotations)
		}
	}
}

// HandlePolicyEvents to process policy related events
func HandlePolicyEvents(data []byte, eventType string, c client.Client) {
	conf, _ := config.ReadConfigs()

	var policyEvent msg.PolicyInfo
	policyEventErr := json.Unmarshal([]byte(string(data)), &policyEvent)
	if policyEventErr != nil {
		logger.LoggerMessaging.Errorf("Error occurred while unmarshalling Throttling Policy event data %v", policyEventErr)
		return
	}
	logger.LoggerMessaging.Infof("===========policy \n%+v\n", policyEvent)
	if strings.EqualFold(eventType, eventConstants.PolicyCreate) {
		if strings.EqualFold(policyEvent.PolicyType, "API") {
			logger.LoggerMessaging.Infof("Policy: %s for policy type: %s for tenant: %s", policyEvent.PolicyName, policyEvent.PolicyType, policyEvent.TenantDomain)
			synchronizer.FetchRateLimitPoliciesOnEvent(policyEvent.PolicyName, policyEvent.TenantDomain, c)
			ratelimitPolicies := managementserver.GetAllRateLimitPolicies()
			logger.LoggerMessaging.Infof("Rate Limit Policies Internal Map: %v", ratelimitPolicies)
		} else if strings.EqualFold(policyEvent.PolicyType, "SUBSCRIPTION") {
			logger.LoggerMessaging.Infof("Policy: %s for policy type: %s", policyEvent.PolicyName, policyEvent.PolicyType)
			synchronizer.FetchSubscriptionRateLimitPoliciesOnEvent(policyEvent.PolicyName, policyEvent.TenantDomain, c, false)
			ratelimitPolicies := managementserver.GetAllRateLimitPolicies()
			logger.LoggerMessaging.Infof("Rate Limit Policies Internal Map: %v", ratelimitPolicies)
		}
	} else if strings.EqualFold(eventType, eventConstants.PolicyUpdate) {
		if strings.EqualFold(policyEvent.PolicyType, "API") {
			logger.LoggerMessaging.Infof("Policy: %s for policy type: %s for tenant: %s", policyEvent.PolicyName, policyEvent.PolicyType, policyEvent.TenantDomain)
			synchronizer.FetchRateLimitPoliciesOnEvent(policyEvent.PolicyName, policyEvent.TenantDomain, c)
			ratelimitPolicies := managementserver.GetAllRateLimitPolicies()
			logger.LoggerMessaging.Infof("Rate Limit Policies Internal Map: %v", ratelimitPolicies)
		} else if strings.EqualFold(policyEvent.PolicyType, "SUBSCRIPTION") {
			logger.LoggerMessaging.Infof("Policy: %s for policy type: %s", policyEvent.PolicyName, policyEvent.PolicyType)
			synchronizer.FetchSubscriptionRateLimitPoliciesOnEvent(policyEvent.PolicyName, policyEvent.TenantDomain, c, false)
			ratelimitPolicies := managementserver.GetAllRateLimitPolicies()
			logger.LoggerMessaging.Infof("Rate Limit Policies Internal Map: %v", ratelimitPolicies)
		}
	} else if strings.EqualFold(eventType, eventConstants.PolicyDelete) {
		if strings.EqualFold(policyEvent.PolicyType, "API") {
			logger.LoggerMessaging.Infof("Policy: %s for policy type: %s", policyEvent.PolicyName, policyEvent.PolicyType)
			managementserver.DeleteRateLimitPolicy(policyEvent.PolicyName, policyEvent.TenantDomain)
			ratelimitPolicies := managementserver.GetAllRateLimitPolicies()
			logger.LoggerMessaging.Infof("Rate Limit Policies Internal Map: %v", ratelimitPolicies)
		} else if strings.EqualFold(policyEvent.PolicyType, "SUBSCRIPTION") {
			logger.LoggerMessaging.Infof("Policy: %s for policy type: %s", policyEvent.PolicyName, policyEvent.PolicyType)
			managementserver.DeleteSubscriptionPolicy(policyEvent.PolicyName, policyEvent.TenantDomain)
			crName := transformer.GeneratePolicyCRName(policyEvent.PolicyName, policyEvent.TenantDomain, "rate-limiting", "subscription")
			internalk8sClient.UnDeployKongPluginCR(crName, c, conf)
			// TODO: undeploy AI ratelimit plugin
			ratelimitPolicies := managementserver.GetAllRateLimitPolicies()
			logger.LoggerMessaging.Infof("Rate Limit Policies Internal Map: %v", ratelimitPolicies)
		}
	}
}

// HandleAIProviderEvents to process AI Provider related events
func HandleAIProviderEvents(data []byte, eventType string, client client.Client) {
	var aiProviderEvent msg.AIProviderEvent
	aiProviderEventErr := json.Unmarshal([]byte(string(data)), &aiProviderEvent)
	if aiProviderEventErr != nil {
		logger.LoggerMessaging.Errorf("Error occurred while unmarshalling AI Provider event data %v", aiProviderEventErr)
		return
	}
	logger.LoggerMessaging.Infof("===========aiprovider \n%+v\n", aiProviderEvent)

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
