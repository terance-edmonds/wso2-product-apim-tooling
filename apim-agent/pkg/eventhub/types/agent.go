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

package types

import (
	"github.com/wso2/product-apim-tooling/apim-agent/config"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Agent defines the functions of a pluggable agent
type Agent interface {
	HandleLifeCycleEvents(data []byte)
	HandleAPIEvents(data []byte, eventType string, conf *config.Config, client client.Client)
	HandleApplicationEvents(data []byte, eventType string)
	HandleSubscriptionEvents(data []byte, eventType string)
	HandlePolicyEvents(data []byte, eventType string, client client.Client)
	HandleAIProviderEvents(data []byte, eventType string, client client.Client)
}
