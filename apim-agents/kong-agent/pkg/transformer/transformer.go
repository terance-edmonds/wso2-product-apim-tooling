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
	"strings"

	v1 "github.com/kong/kubernetes-configuration/api/configuration/v1"
	"github.com/terance-edmonds/wso2-apk-k8s-go-lib/config/constants"
	"github.com/terance-edmonds/wso2-apk-k8s-go-lib/config/types"
	httpGenerator "github.com/terance-edmonds/wso2-apk-k8s-go-lib/pkg/generators/http"
	"github.com/terance-edmonds/wso2-apk-k8s-go-lib/pkg/utils"
	eventHub "github.com/wso2/product-apim-tooling/apim-agent/pkg/eventhub/types"
	apimTransformer "github.com/wso2/product-apim-tooling/apim-agent/pkg/transformer"
	logger "github.com/wso2/product-apim-tooling/apim-agents/kong-agent/internal/loggers"
	pkgConstants "github.com/wso2/product-apim-tooling/apim-agents/kong-agent/pkg/constants"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// GenerateCR handles the generation k8s artifacts
func GenerateCR(api string, organizationID string, apiUUID string) *K8sArtifacts {
	k8sArtifact := K8sArtifacts{APIUUID: apiUUID, HTTPRoutes: make(map[string]*gwapiv1.HTTPRoute), Services: make(map[string]*corev1.Service), KongPlugins: map[string]*v1.KongPlugin{}}
	kongPlugins := []string{}
	var apkConf types.APKConf
	err := yaml.Unmarshal([]byte(api), &apkConf)
	if err != nil {
		logger.LoggerUtils.Errorf("Error while converting apk conf yaml to apk conf type: Error: %+v. \n", err)
	}
	apiUniqueID := GetUniqueIDForAPI(apkConf.Name, apkConf.Version, organizationID)

	// create endpoints
	createdEndpoints := utils.GetEndpoints(apkConf)

	// handle authentications
	authentications := *apkConf.Authentication
	for _, authentication := range authentications {
		if !authentication.Enabled {
			continue
		}

		// OAuth2 JWT Plugin (for OAuth2 jwt authentication)
		if authentication.AuthType == pkgConstants.OAuth2 {
			kongJwtPlugin := createAndAddJWTPlugin(&k8sArtifact, nil, "api")
			// * Only in Kong Enterprise
			// kongJwtPlugin.Ordering = &kong.PluginOrdering{
			// 	After: map[string][]string{
			// 		"access": {"acl"},
			// 	},
			// }

			kongPlugins = append(kongPlugins, kongJwtPlugin.ObjectMeta.Name)
		}
	}

	// create ratelimit policies
	if apkConf.RateLimit != nil {
		rateLimitConfig := KongPluginConfig{
			"limit_by": "service",
		}
		PrepareRateLimit(&rateLimitConfig, apkConf.RateLimit.Unit, apkConf.RateLimit.RequestsPerUnit)
		kongRateLimitPlugin := GenerateRateLimitPlugin(nil, "api", rateLimitConfig)

		k8sArtifact.KongPlugins[kongRateLimitPlugin.ObjectMeta.Name] = kongRateLimitPlugin
		kongPlugins = append(kongPlugins, kongRateLimitPlugin.ObjectMeta.Name)
	}

	// HTTPRoute
	// generate production http routes
	if endpoints, ok := createdEndpoints[constants.PRODUCTION_TYPE]; ok {
		generateHTTPRoutes(&k8sArtifact, &apkConf, organizationID, endpoints, constants.PRODUCTION_TYPE, apiUniqueID, kongPlugins)
	}
	// generate sandbox http routes
	if endpoints, ok := createdEndpoints[constants.SANDBOX_TYPE]; ok {
		generateHTTPRoutes(&k8sArtifact, &apkConf, organizationID, endpoints, constants.SANDBOX_TYPE, apiUniqueID, kongPlugins)
	}

	return &k8sArtifact
}

// UpdateCRS cr update
func UpdateCRS(k8sArtifact *K8sArtifacts, environments *[]apimTransformer.Environment, organizationID string, apiUUID string, revisionID string, namespace string, configuredRateLimitPoliciesMap map[string]eventHub.RateLimitPolicy) {
	organizationHash := generateSHA1Hash(organizationID)

	// generate Cors Configurations for the gateway
	origins := []string{}
	logger.LoggerMessaging.Infof("env :\n%+v\n", environments)
	for _, environment := range *environments {
		vhost := environment.Vhost
		origins = append(origins, vhost)
	}
	corsConfig := KongPluginConfig{
		"origins": origins,
	}
	corsPlugin := GenerateCorsPlugin(nil, "api", corsConfig)
	k8sArtifact.KongPlugins[corsPlugin.ObjectMeta.Name] = corsPlugin

	for _, httproute := range k8sArtifact.HTTPRoutes {
		// TODO: add cors plugin to httproute
		httproute.ObjectMeta.Labels[k8sOrganizationField] = organizationHash
		httproute.ObjectMeta.Labels[k8sAPIUuidField] = apiUUID
		httproute.ObjectMeta.Labels[k8sRevisionField] = revisionID
		// update hostnames
		for _, environment := range *environments {
			vhost := environment.Vhost
			if httproute.ObjectMeta.Labels[k8sAPIEnvironmentField] == constants.PRODUCTION_TYPE {
				httproute.Spec.Hostnames = []gwapiv1.Hostname{gwapiv1.Hostname(vhost)}
			}
			if httproute.ObjectMeta.Labels[k8sAPIEnvironmentField] == constants.SANDBOX_TYPE {
				httproute.Spec.Hostnames = []gwapiv1.Hostname{gwapiv1.Hostname("sandbox." + vhost)}
			}
		}
	}
	for _, service := range k8sArtifact.Services {
		service.ObjectMeta.Labels = make(map[string]string)
		service.ObjectMeta.Labels[k8sOrganizationField] = organizationHash
		service.ObjectMeta.Labels[k8sAPIUuidField] = apiUUID
		service.ObjectMeta.Labels[k8sRevisionField] = revisionID
	}
	for _, kongPlugin := range k8sArtifact.KongPlugins {
		kongPlugin.ObjectMeta.Labels = make(map[string]string)
		kongPlugin.ObjectMeta.Labels[k8sOrganizationField] = organizationHash
		kongPlugin.ObjectMeta.Labels[k8sAPIUuidField] = apiUUID
		kongPlugin.ObjectMeta.Labels[k8sRevisionField] = revisionID
	}
}

// generateHTTPRoutes handles the generation of http route resources from apk conf
func generateHTTPRoutes(k8sArtifact *K8sArtifacts, apkConf *types.APKConf, organizationID string, endpoints types.EndpointDetails, endpointType string, uniqueID string, kongPlugins []string) {
	// ACL Plugin (for subscription)
	// create and add route restriction with Kong ACL plugin into k8s artifacts
	if apkConf.SubscriptionValidation {
		kongACLPlugin := createAndAddACLPlugin(k8sArtifact, nil, "api", endpointType)
		// * Only in Kong Enterprise
		// kongACLPlugin.Ordering = &kong.PluginOrdering{
		// 	Before: map[string][]string{
		// 		"access": {"jwt"},
		// 	},
		// }

		kongPlugins = append(kongPlugins, kongACLPlugin.ObjectMeta.Name)
	}

	gen := httpGenerator.Generator()
	organization := types.Organization{
		Name: organizationID,
	}
	gatewayConfigurations := types.GatewayConfigurations{
		Name: k8sIngressClassName,
	}

	operationsArray := GenerateOperationsMatrix(len(*apkConf.Operations), 8)
	row := 0
	column := 0
	for i := 0; i < len(*apkConf.Operations); i++ {
		if column > 7 {
			row++
			column = 0
		}
		operationsArray[row][column] = (*apkConf.Operations)[i]
		column++
	}

	for i, operations := range operationsArray {
		logger.LoggerUtils.Infof("Generate Operations: %v\n", operations)
		httpK8sArtifact, err := gen.GenerateHTTPRoute(*apkConf, organization, gatewayConfigurations, operations, &endpoints, endpointType, uniqueID, i)
		if err != nil {
			logger.LoggerUtils.Errorf("Error while generating http route: Error: %+v. \n", err)
		} else {
			httpRoute := httpK8sArtifact.HTTPRoute
			httpRoute.Spec.ParentRefs[0].SectionName = nil
			// initialize labels structure and add environment type
			if httpRoute.ObjectMeta.Labels == nil {
				httpRoute.ObjectMeta.Labels = make(map[string]string)
			}
			httpRoute.ObjectMeta.Labels[k8sAPIEnvironmentField] = endpointType

			// store the services into k8s artifacts
			for key, service := range httpK8sArtifact.Services {
				k8sArtifact.Services[key] = service
			}

			// update httproute annotation
			annotationMap := map[string]string{
				"konghq.com/strip-path": "true",
				"konghq.com/plugins":    strings.Join(kongPlugins, ","),
			}
			updateHTTPRouteAnnotations(httpRoute, annotationMap)

			// store httproute in k8s artifacts
			k8sArtifact.HTTPRoutes[httpRoute.ObjectMeta.Name] = httpRoute
		}
	}
}

// createAndAddACLPlugin handles the Kong ACL credential plugin generation and adding to k8s resources
func createAndAddACLPlugin(k8sArtifact *K8sArtifacts, operation *types.Operation, targetRef string, environment string) *v1.KongPlugin {
	productionGroupName := GenerateACLGroupName(k8sArtifact.APIUUID, environment)
	config := KongPluginConfig{
		"allow": []string{
			productionGroupName,
		},
	}
	targetRef = targetRef + "-" + environment
	aclPlugin := GenerateACLPlugin(operation, targetRef, config)
	k8sArtifact.KongPlugins[aclPlugin.ObjectMeta.Name] = aclPlugin
	return aclPlugin
}

// createAndAddJWTPlugin handles the Kong JWT credential plugin generation and adding to k8s resources
func createAndAddJWTPlugin(k8sArtifact *K8sArtifacts, operation *types.Operation, targetRef string) *v1.KongPlugin {
	config := KongPluginConfig{
		"run_on_preflight": true,
		"key_claim_name":   "client_id",
		"header_names": []string{
			"authorization",
		},
	}
	jwtPlugin := GenerateJWTPlugin(operation, targetRef, config)
	k8sArtifact.KongPlugins[jwtPlugin.ObjectMeta.Name] = jwtPlugin
	return jwtPlugin
}

// updateHTTPRouteAnnotations updates the annotations of httproutes
func updateHTTPRouteAnnotations(httpRoute *gwapiv1.HTTPRoute, annotations map[string]string) {
	httpRoute.Annotations = make(map[string]string, 0)
	for key, annotation := range annotations {
		httpRoute.Annotations[key] = strings.Join([]string{annotation, httpRoute.Annotations[key]}, ",")
	}
}

// CreateConsumer handles the Kong consumer generation
func CreateConsumer(subscriptionUUID string, applicationUUID string, apiUUID string, environment string) *v1.KongConsumer {
	consumer := v1.KongConsumer{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KongConsumer",
			APIVersion: "configuration.konghq.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: GenerateConsumerName(subscriptionUUID, applicationUUID, apiUUID, environment),
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": k8sIngressClassName,
			},
			Labels: make(map[string]string, 0),
		},
		Username: generateSHA1Hash(subscriptionUUID + apiUUID + applicationUUID + environment),
		CustomID: generateSHA1Hash(subscriptionUUID + applicationUUID + environment),
	}
	consumer.Labels[k8APPUuidField] = applicationUUID
	consumer.Labels[k8sAPIUuidField] = apiUUID
	consumer.Labels[k8sAPIEnvironmentField] = environment
	return &consumer
}

// GenerateK8sCredentialSecret handles the k8s secret generation for kong credentials
func GenerateK8sCredentialSecret(applicationUUID string, identifier string, credentialName string, data map[string]string) *corev1.Secret {
	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: GenerateSecretName(applicationUUID, identifier, credentialName),
			Labels: map[string]string{
				"konghq.com/credential": credentialName,
			},
		},
		StringData: data,
	}
	secret.Labels[k8APPUuidField] = applicationUUID
	return &secret
}

// GenerateK8sSecret handles the k8s secret generation
func GenerateK8sSecret(name string, labels map[string]string, data map[string]string) *corev1.Secret {
	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   PrepareSecretName(name),
			Labels: labels,
		},
		StringData: data,
	}
	return &secret
}
