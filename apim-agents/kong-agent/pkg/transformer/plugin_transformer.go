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

package transformer

import (
	v1 "github.com/kong/kubernetes-configuration/api/configuration/v1"
	"github.com/terance-edmonds/wso2-apk-k8s-go-lib/config/types"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GenerateACLPlugin handles the Kong ACL plugin generation
func GenerateACLPlugin(operation *types.Operation, targetRef string, config KongPluginConfig) *v1.KongPlugin {
	return &v1.KongPlugin{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KongPlugin",
			APIVersion: "configuration.konghq.com/v1",
		},
		PluginName: "acl",
		ObjectMeta: metav1.ObjectMeta{
			Name: GeneratePluginRefName(operation, targetRef, "acl"),
		},
		Config: apiextensionsv1.JSON{
			Raw: GenerateJSON(config),
		},
	}
}

// GenerateJWTPlugin handles the Kong JWT plugin generation
func GenerateJWTPlugin(operation *types.Operation, targetRef string, config KongPluginConfig) *v1.KongPlugin {
	return &v1.KongPlugin{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KongPlugin",
			APIVersion: "configuration.konghq.com/v1",
		},
		PluginName: "jwt",
		ObjectMeta: metav1.ObjectMeta{
			Name: GeneratePluginRefName(operation, targetRef, "jwt"),
		},
		Config: apiextensionsv1.JSON{
			Raw: GenerateJSON(config),
		},
	}
}

// GenerateRateLimitPlugin handles the Kong RateLimit plugin generation
func GenerateRateLimitPlugin(operation *types.Operation, targetRef string, config KongPluginConfig) *v1.KongPlugin {
	return &v1.KongPlugin{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KongPlugin",
			APIVersion: "configuration.konghq.com/v1",
		},
		PluginName: "rate-limiting",
		ObjectMeta: metav1.ObjectMeta{
			Name: GeneratePluginRefName(operation, targetRef, "rate-limiting"),
		},
		Config: apiextensionsv1.JSON{
			Raw: GenerateJSON(config),
		},
	}
}
