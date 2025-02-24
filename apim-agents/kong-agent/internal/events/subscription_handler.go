package events

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/terance-edmonds/wso2-apk-k8s-go-lib/config/constants"
	"github.com/wso2/product-apim-tooling/apim-agent/config"
	eventConstants "github.com/wso2/product-apim-tooling/apim-agent/pkg/eventhub/constants"
	"github.com/wso2/product-apim-tooling/apim-agent/pkg/managementserver"
	msg "github.com/wso2/product-apim-tooling/apim-agent/pkg/messaging"
	internalk8sClient "github.com/wso2/product-apim-tooling/apim-agents/kong-agent/internal/k8sClient"
	logger "github.com/wso2/product-apim-tooling/apim-agents/kong-agent/internal/loggers"
	"github.com/wso2/product-apim-tooling/apim-agents/kong-agent/pkg/transformer"
	"github.com/wso2/product-apim-tooling/apim-agents/kong-agent/pkg/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// HandleSubscriptionEvents to process subscription related events
func HandleSubscriptionEvents(data []byte, eventType string, c client.Client) {
	conf, _ := config.ReadConfigs()

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
		// create production consumer and acl credential
		createSubscription(subscriptionEvent, c, conf, constants.PRODUCTION_TYPE)
		// create sandbox consumer and acl credential
		createSubscription(subscriptionEvent, c, conf, constants.SANDBOX_TYPE)
	} else if subscriptionEvent.Event.Type == eventConstants.SubscriptionUpdate {
		// update production consumer and configurations
		updateSubscription(subscriptionEvent, c, conf, constants.PRODUCTION_TYPE)
		// update sandbox consumer and configurations
		updateSubscription(subscriptionEvent, c, conf, constants.SANDBOX_TYPE)
	} else if subscriptionEvent.Event.Type == eventConstants.SubscriptionDelete {
		// remove production ACL credentials and consumer
		removeSubscription(subscriptionEvent, c, conf, constants.PRODUCTION_TYPE)
		// remove sandbox ACL credentials and consumer
		removeSubscription(subscriptionEvent, c, conf, constants.SANDBOX_TYPE)
	}
}

func createSubscription(subscriptionEvent msg.SubscriptionEvent, c client.Client, conf *config.Config, environment string) {
	consumer := transformer.CreateConsumer(subscriptionEvent.SubscriptionUUID, subscriptionEvent.ApplicationUUID, subscriptionEvent.APIUUID, environment)
	consumer.Namespace = conf.DataPlane.Namespace

	// create kong acl secret CR
	aclCredentialSecretConfig := map[string]string{
		"group": transformer.GenerateACLGroupName(subscriptionEvent.APIUUID, environment),
	}
	subscriptionIdentifier := subscriptionEvent.APIUUID + environment
	aclCredentialSecret := transformer.GenerateK8sCredentialSecret(subscriptionEvent.ApplicationUUID, subscriptionIdentifier, "acl", aclCredentialSecretConfig)
	aclCredentialSecret.Namespace = conf.DataPlane.Namespace
	credentials := []string{aclCredentialSecret.ObjectMeta.Name}

	// update consumer subscription limit plugin annotation
	subscriptionPolicy := managementserver.GetSubscriptionPolicy(subscriptionEvent.PolicyID, subscriptionEvent.TenantDomain)
	logger.LoggerMessaging.Infof("Subscription Policy: %v", subscriptionPolicy)
	if subscriptionPolicy.Name != "" && subscriptionPolicy.Name != "Unlimited" {
		rateLimitCRName := transformer.GeneratePolicyCRName(subscriptionPolicy.Name, subscriptionPolicy.TenantDomain, "rate-limiting", "subscription")
		addAnnotations := []string{rateLimitCRName}

		consumer.Annotations["konghq.com/plugins"] = utils.PrepareAnnotations(consumer.Annotations["konghq.com/plugins"], addAnnotations, nil)
	}

	// get available jwt credentials for the application
	jwtSecretCredentials := internalk8sClient.GetK8sSecrets(map[string]string{
		"applicationUUID":       subscriptionEvent.ApplicationUUID,
		"environment":           environment,
		"konghq.com/credential": "jwt",
	}, c, conf)
	for _, jwtSecretCredential := range jwtSecretCredentials {
		credentials = append(credentials, jwtSecretCredential.Name)
	}

	// update consumers credentials
	consumer.Credentials = utils.AddItems(consumer.Credentials, credentials)
	// deploy acl secret
	internalk8sClient.DeploySecretCR(aclCredentialSecret, c)
	// deploy consumer
	internalk8sClient.DeployKongConsumerCR(consumer, c)
}

func updateSubscription(subscriptionEvent msg.SubscriptionEvent, c client.Client, conf *config.Config, environment string) {
	var removeAnnotations []string
	var addAnnotations []string
	// retrieving current production subscription policy
	consumerName := transformer.GenerateConsumerName(subscriptionEvent.SubscriptionUUID, subscriptionEvent.ApplicationUUID, subscriptionEvent.APIUUID, environment)
	consumer := internalk8sClient.GetKongConsumerCR(consumerName, c, conf)

	if consumer == nil {
		logger.LoggerMessaging.Infof("Kong consumer credential not found for %v", environment)
	} else {
		subscriptionPolicy := managementserver.GetSubscriptionPolicy(subscriptionEvent.PolicyID, subscriptionEvent.TenantDomain)
		rateLimitCRName := transformer.GeneratePolicyCRName(subscriptionPolicy.Name, subscriptionPolicy.TenantDomain, "rate-limiting", "subscription")
		// handle subscription rate limiting
		if annotations, ok := consumer.Annotations["konghq.com/plugins"]; ok {
			annotationsArr := strings.Split(annotations, ",")
			if !slices.Contains(annotationsArr, rateLimitCRName) {
				// remove old subscription policy name
				for _, name := range annotationsArr {
					if strings.Contains(name, "subscription") && strings.Contains(name, "rate-limiting") {
						removeAnnotations = append(removeAnnotations, name)
						break
					}
				}

				// updating new subscription policy name
				if subscriptionPolicy.Name != "" && subscriptionPolicy.Name != "Unlimited" {
					addAnnotations = append(addAnnotations, rateLimitCRName)
				}

				internalk8sClient.UpdateKongConsumerPluginAnnotation(subscriptionEvent.ApplicationUUID, subscriptionEvent.APIUUID, c, conf, addAnnotations, removeAnnotations)
			}
		}

		// handle subscription state
		aclCredentialSecretName := transformer.GenerateSecretName(subscriptionEvent.ApplicationUUID, subscriptionEvent.APIUUID, "acl")
		credentials := []string{aclCredentialSecretName}
		if subscriptionEvent.SubscriptionState == "BLOCKED" {
			internalk8sClient.UpdateKongConsumerCredential(subscriptionEvent.ApplicationUUID, constants.SANDBOX_TYPE, c, conf, nil, credentials)
			internalk8sClient.UpdateKongConsumerCredential(subscriptionEvent.ApplicationUUID, constants.PRODUCTION_TYPE, c, conf, nil, credentials)
		} else if subscriptionEvent.SubscriptionState == "PROD_ONLY_BLOCKED" {
			internalk8sClient.UpdateKongConsumerCredential(subscriptionEvent.ApplicationUUID, constants.SANDBOX_TYPE, c, conf, credentials, nil)
			internalk8sClient.UpdateKongConsumerCredential(subscriptionEvent.ApplicationUUID, constants.PRODUCTION_TYPE, c, conf, nil, credentials)
		} else if subscriptionEvent.SubscriptionState == "UNBLOCKED" {
			internalk8sClient.UpdateKongConsumerCredential(subscriptionEvent.ApplicationUUID, constants.SANDBOX_TYPE, c, conf, credentials, nil)
			internalk8sClient.UpdateKongConsumerCredential(subscriptionEvent.ApplicationUUID, constants.PRODUCTION_TYPE, c, conf, credentials, nil)
		}
	}

}

func removeSubscription(subscriptionEvent msg.SubscriptionEvent, c client.Client, conf *config.Config, environment string) {
	// remove acl secret credential
	subscriptionIdentifier := subscriptionEvent.APIUUID + environment
	aclSecretCredentialName := transformer.GenerateSecretName(subscriptionEvent.ApplicationUUID, subscriptionIdentifier, "acl")
	internalk8sClient.UnDeploySecretCR(aclSecretCredentialName, c, conf)

	// remove kong consumer CR
	consumerName := transformer.GenerateConsumerName(subscriptionEvent.SubscriptionUUID, subscriptionEvent.ApplicationUUID, subscriptionEvent.APIUUID, environment)
	internalk8sClient.UnDeployKongConsumerCR(consumerName, c, conf)
}
