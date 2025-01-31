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
	"crypto/sha1"
	"encoding/hex"

	"github.com/terance-edmonds/wso2-apk-k8s-go-lib/config/constants"
	"github.com/terance-edmonds/wso2-apk-k8s-go-lib/config/types"
	http_generator "github.com/terance-edmonds/wso2-apk-k8s-go-lib/pkg/generators/http"
	"github.com/terance-edmonds/wso2-apk-k8s-go-lib/pkg/utils"
	eventHub "github.com/wso2/product-apim-tooling/apim-agent/pkg/eventhub/types"
	apimTransformer "github.com/wso2/product-apim-tooling/apim-agent/pkg/transformer"
	logger "github.com/wso2/product-apim-tooling/apim-agents/kong-agent/internal/loggers"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// GenerateCR handles the generation k8s artifacts
func GenerateCR(api string, organizationID string) *K8sArtifacts {
	k8sArtifact := K8sArtifacts{HTTPRoutes: make(map[string]*gwapiv1.HTTPRoute), Services: make(map[string]*corev1.Service)}
	var apkConf types.APKConf
	err := yaml.Unmarshal([]byte(api), &apkConf)
	if err != nil {
		logger.LoggerUtils.Errorf("Error while converting apk conf yaml to apk conf type: Error: %+v. \n", err)
	}
	apiUniqueID := GetUniqueIDForAPI(apkConf.Name, apkConf.Version, organizationID)

	// Create endpoints
	createdEndpoints := utils.GetEndpoints(apkConf)

	// HTTPRoute
	// Generate production http routes
	if endpoints, ok := createdEndpoints[constants.PRODUCTION_TYPE]; ok {
		generateHTTPRoutes(&k8sArtifact, &apkConf, organizationID, endpoints, constants.PRODUCTION_TYPE, apiUniqueID)
	}
	// Generate sandbox http routes
	if endpoints, ok := createdEndpoints[constants.SANDBOX_TYPE]; ok {
		generateHTTPRoutes(&k8sArtifact, &apkConf, organizationID, endpoints, constants.SANDBOX_TYPE, apiUniqueID)
	}

	return &k8sArtifact
}

// generateHTTPRoutes handles the generation of http route resources from apk conf
func generateHTTPRoutes(k8sArtifact *K8sArtifacts, apkConf *types.APKConf, organizationID string, endpoints types.EndpointDetails, endpointType string, uniqueID string) {
	gen := http_generator.Generator()
	organization := types.Organization{
		Name: organizationID,
	}
	gatewayConfigurations := types.GatewayConfigurations{
		Name:         "kong",
		ListenerName: "http",
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
			k8sArtifact.HTTPRoutes[httpRoute.ObjectMeta.Name] = httpRoute
			for key, service := range httpK8sArtifact.Services {
				k8sArtifact.Services[key] = service
			}
		}
	}
}

// UpdateCRS cr update
func UpdateCRS(k8sArtifact *K8sArtifacts, environments *[]apimTransformer.Environment, organizationID string, apiUUID string, revisionID string, namespace string, configuredRateLimitPoliciesMap map[string]eventHub.RateLimitPolicy) {
	organizationHash := generateSHA1Hash(organizationID)
	for _, httproute := range k8sArtifact.HTTPRoutes {
		httproute.ObjectMeta.Labels = make(map[string]string)
		httproute.ObjectMeta.Labels[k8sOrganizationField] = organizationHash
		httproute.ObjectMeta.Labels[k8APIUuidField] = apiUUID
		httproute.ObjectMeta.Labels[k8RevisionField] = revisionID
	}
	for _, service := range k8sArtifact.Services {
		service.ObjectMeta.Labels = make(map[string]string)
		service.ObjectMeta.Labels[k8sOrganizationField] = organizationHash
		service.ObjectMeta.Labels[k8APIUuidField] = apiUUID
		service.ObjectMeta.Labels[k8RevisionField] = revisionID
	}
}

// generateSHA1Hash returns the SHA1 hash for the given string
func generateSHA1Hash(input string) string {
	h := sha1.New() /* #nosec */
	h.Write([]byte(input))
	return hex.EncodeToString(h.Sum(nil))
}
