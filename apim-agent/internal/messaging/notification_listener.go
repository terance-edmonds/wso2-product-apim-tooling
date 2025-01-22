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

// Package messaging holds the implementation for event listeners functions
package messaging

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/wso2/product-apim-tooling/apim-agent/config"
	logger "github.com/wso2/product-apim-tooling/apim-agent/internal/loggers"
	"github.com/wso2/product-apim-tooling/apim-agent/pkg/agent"
	"github.com/wso2/product-apim-tooling/apim-agent/pkg/eventhub/types"
	msg "github.com/wso2/product-apim-tooling/apim-agent/pkg/messaging"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// constant variables
const (
	apiEventType                = "API"
	applicationEventType        = "APPLICATION"
	subscriptionEventType       = "SUBSCRIPTION"
	scopeEvenType               = "SCOPE"
	policyEventType             = "POLICY"
	removeAPIFromGateway        = "REMOVE_API_FROM_GATEWAY"
	deployAPIToGateway          = "DEPLOY_API_IN_GATEWAY"
	applicationRegistration     = "APPLICATION_REGISTRATION_CREATE"
	removeApplicationKeyMapping = "REMOVE_APPLICATION_KEYMAPPING"
	apiLifeCycleChange          = "LIFECYCLE_CHANGE"
	applicationCreate           = "APPLICATION_CREATE"
	applicationUpdate           = "APPLICATION_UPDATE"
	applicationDelete           = "APPLICATION_DELETE"
	subscriptionCreate          = "SUBSCRIPTIONS_CREATE"
	subscriptionUpdate          = "SUBSCRIPTIONS_UPDATE"
	subscriptionDelete          = "SUBSCRIPTIONS_DELETE"
	policyCreate                = "POLICY_CREATE"
	policyUpdate                = "POLICY_UPDATE"
	policyDelete                = "POLICY_DELETE"
	blockedStatus               = "BLOCKED"
	apiUpdate                   = "API_UPDATE"
	aiProviderEventType         = "LLM_PROVIDER"
	aiProviderCreate            = "LLM_PROVIDER_CREATE"
	aiProviderUpdate            = "LLM_PROVIDER_UPDATE"
	aiProviderDelete            = "LLM_PROVIDER_DELETE"
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

// handleNotification to process
func handleNotification(c client.Client, agent agent.Agent) {
	conf, _ := config.ReadConfigs()
	for d := range msg.NotificationChannel {
		var notification msg.EventNotification
		notificationErr := parseNotificationJSONEvent([]byte(string(d.Body)), &notification)
		if notificationErr != nil {
			continue
		}
		logger.LoggerMessaging.Infof("Event %s is received", notification.Event.PayloadData.EventType)
		logger.LoggerMessaging.Infof("Event %s is received with payload %s", notification.Event.PayloadData.EventType, notification.Event.PayloadData.Event)
		err := processNotificationEvent(conf, &notification, c, agent)
		if err != nil {
			continue
		}
		d.Ack(false)
	}
	logger.LoggerMessaging.Infof("handle: deliveries channel closed")
}

func processNotificationEvent(conf *config.Config, notification *msg.EventNotification, c client.Client, agent agent.Agent) error {
	var eventType string
	var decodedByte, err = base64.StdEncoding.DecodeString(notification.Event.PayloadData.Event)
	if err != nil {
		if _, ok := err.(base64.CorruptInputError); ok {
			logger.LoggerMessaging.Error("\nbase64 input is corrupt, check the provided key")
		}
		logger.LoggerMessaging.Errorf("Error occurred while decoding the notification event %v. "+
			"Hence dropping the event", err)
		return err
	}

	AgentMode := conf.Agent.Mode
	eventType = notification.Event.PayloadData.EventType
	if strings.Contains(eventType, apiLifeCycleChange) {
		if AgentMode == "CPtoDP" {
			agent.HandleLifeCycleEvents(decodedByte)
		}
	} else if strings.Contains(eventType, apiEventType) {
		if AgentMode == "CPtoDP" {
			agent.HandleAPIEvents(decodedByte, eventType, conf, c)
		}
	} else if strings.Contains(eventType, applicationEventType) {
		agent.HandleApplicationEvents(decodedByte, eventType)
	} else if strings.Contains(eventType, subscriptionEventType) {
		agent.HandleSubscriptionEvents(decodedByte, eventType)
	} else if strings.Contains(eventType, policyEventType) {
		var policyEvent msg.PolicyInfo
		policyEventErr := json.Unmarshal([]byte(string(decodedByte)), &policyEvent)
		if policyEventErr != nil {
			logger.LoggerMessaging.Errorf("Error occurred while unmarshalling Throttling Policy event data %v", policyEventErr)
		}
		if AgentMode == "CPtoDP" || strings.EqualFold(policyEvent.PolicyType, "SUBSCRIPTION") {
			agent.HandlePolicyEvents(decodedByte, eventType, c)
		}
	} else if strings.Contains(eventType, aiProviderEventType) {
		agent.HandleAIProviderEvents(decodedByte, eventType, c)
	}
	// other events will ignore including HEALTH_CHECK event
	return nil
}

func parseNotificationJSONEvent(data []byte, notification *msg.EventNotification) error {
	unmarshalErr := json.Unmarshal(data, &notification)
	if unmarshalErr != nil {
		logger.LoggerMessaging.Errorf("Error occurred while unmarshalling "+
			"notification event data %v. Hence dropping the event", unmarshalErr)
	}
	return unmarshalErr
}
