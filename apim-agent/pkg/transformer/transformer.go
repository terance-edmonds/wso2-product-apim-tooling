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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/wso2/product-apim-tooling/apim-agent/internal/constants"
	eventHub "github.com/wso2/product-apim-tooling/apim-agent/pkg/eventhub/types"
	"github.com/wso2/product-apim-tooling/apim-agent/pkg/managementserver"

	logger "github.com/wso2/product-apim-tooling/apim-agent/pkg/loggers"
	"gopkg.in/yaml.v2"
)

// GenerateConf will Generate the mapped .apk-conf file for a given API Project zip
func GenerateConf(APIJson string, certArtifact CertificateArtifact, organizationID string) (string, string, uint32, map[string]eventHub.RateLimitPolicy, EndpointSecurityConfig, *API, *AIRatelimit, *AIRatelimit, error) {

	apk := &API{}

	var apiYaml APIYaml

	var configuredRateLimitPoliciesMap = make(map[string]eventHub.RateLimitPolicy)

	logger.LoggerTransformer.Debugf("APIJson: %v", APIJson)

	apiYamlError := json.Unmarshal([]byte(APIJson), &apiYaml)

	if apiYamlError != nil {
		logger.LoggerTransformer.Error("Error while unmarshalling api.json content", apiYamlError)
		return "", "null", 0, nil, EndpointSecurityConfig{}, nil, nil, nil, apiYamlError
	}

	apiYamlData := apiYaml.Data
	logger.LoggerTransformer.Debugf("apiYamlData: %v", apiYamlData)

	apk.Name = apiYamlData.Name
	apk.Context = apiYamlData.Context
	apk.Version = apiYamlData.Version
	apk.Type = getAPIType(apiYamlData.Type)
	apk.DefaultVersion = apiYamlData.DefaultVersion
	apk.DefinitionPath = "/definition"
	apk.SubscriptionValidation = true

	if apiYamlData.SubtypeConfiguration.Subtype == "AIAPI" && apiYamlData.SubtypeConfiguration.Configuration != "" {
		// Unmarshal the _configuration field into the Configuration struct
		var config Configuration
		err := json.Unmarshal([]byte(apiYamlData.SubtypeConfiguration.Configuration), &config)
		if err != nil {
			fmt.Println("Error unmarshalling _configuration:", err)
			return "", "null", 0, nil, EndpointSecurityConfig{}, nil, nil, nil, err
		}
		sha1ValueforCRName := config.LLMProviderID
		apk.AIProvider = &AIProvider{
			Name:       sha1ValueforCRName,
			APIVersion: "1",
		}
	}

	if apiYamlData.APIThrottlingPolicy != "" {
		rateLimitPolicy := managementserver.GetRateLimitPolicy(apiYamlData.APIThrottlingPolicy, organizationID)
		logger.LoggerTransformer.Debugf("Rate Limit Policy: %v", rateLimitPolicy)
		if rateLimitPolicy.Name != "" && rateLimitPolicy.Name != "Unlimited" {
			var rateLimitPolicyConfigured = RateLimit{
				RequestsPerUnit: rateLimitPolicy.DefaultLimit.RequestCount.RequestCount,
				Unit:            rateLimitPolicy.DefaultLimit.RequestCount.TimeUnit,
			}
			apk.RateLimit = &rateLimitPolicyConfigured
			configuredRateLimitPoliciesMap["API"] = rateLimitPolicy
		}
	}
	apkOperations := make([]Operation, len(apiYamlData.Operations))

	for i, operation := range apiYamlData.Operations {

		reqPolicyCount := len(operation.OperationPolicies.Request)
		resPolicyCount := len(operation.OperationPolicies.Response)
		reqInterceptor, resInterceptor := getReqAndResInterceptors(reqPolicyCount, resPolicyCount,
			operation.OperationPolicies.Request, operation.OperationPolicies.Response)

		var opRateLimit *RateLimit
		if apiYamlData.APIThrottlingPolicy == "" && operation.ThrottlingPolicy != "" {
			rateLimitPolicy := managementserver.GetRateLimitPolicy(operation.ThrottlingPolicy, organizationID)
			logger.LoggerTransformer.Debugf("Op Rate Limit Policy Name: %v", rateLimitPolicy.Name)
			if rateLimitPolicy.Name != "" && rateLimitPolicy.Name != "Unlimited" {
				var rateLimitPolicyConfigured = RateLimit{
					RequestsPerUnit: rateLimitPolicy.DefaultLimit.RequestCount.RequestCount,
					Unit:            rateLimitPolicy.DefaultLimit.RequestCount.TimeUnit,
				}
				opRateLimit = &rateLimitPolicyConfigured
				configuredRateLimitPoliciesMap["Resource"] = rateLimitPolicy
			}
		}
		logger.LoggerTransformer.Debugf("Operation Auth Type: %v", operation.AuthType)
		AuthSecured := true
		if operation.AuthType == "None" {
			logger.LoggerTransformer.Debugf("Setting AuthSecured to false")
			AuthSecured = false
		}
		op := &Operation{
			Target:  operation.Target,
			Verb:    operation.Verb,
			Scopes:  operation.Scopes,
			Secured: AuthSecured,
			OperationPolicies: &OperationPolicies{
				Request:  *reqInterceptor,
				Response: *resInterceptor,
			},
			RateLimit: opRateLimit,
		}
		apkOperations[i] = *op
	}

	apk.Operations = &apkOperations

	//Adding API Level Operation Policies to the conf
	reqPolicyCount := len(apiYaml.Data.APIPolicies.Request)
	resPolicyCount := len(apiYaml.Data.APIPolicies.Response)
	reqInterceptor, resInterceptor := getReqAndResInterceptors(reqPolicyCount, resPolicyCount,
		apiYaml.Data.APIPolicies.Request, apiYaml.Data.APIPolicies.Response)

	apk.APIPolicies = &OperationPolicies{
		Request:  *reqInterceptor,
		Response: *resInterceptor,
	}

	//Adding Endpoint-certificate configurations to the conf
	var endpointCertList EndpointCertDescriptor
	endCertAvailable := false

	if certArtifact.EndpointCerts != "" {
		certErr := json.Unmarshal([]byte(certArtifact.EndpointCerts), &endpointCertList)
		if certErr != nil {
			logger.LoggerTransformer.Errorf("Error while unmarshalling endpoint_cert.json content: %v", apiYamlError)
			return "", "null", 0, nil, EndpointSecurityConfig{}, nil, nil, nil, certErr
		}
		endCertAvailable = true
	}

	sandboxURL := apiYamlData.EndpointConfig.SandboxEndpoints.URL
	prodURL := apiYamlData.EndpointConfig.ProductionEndpoints.URL
	endpointSecurityData := apiYamlData.EndpointConfig.EndpointSecurity
	apiUniqueID := GetUniqueIDForAPI(apiYamlData.Name, apiYamlData.Version, apiYamlData.OrganizationID)
	logger.LoggerTransformer.Infof("Maxtps: %+v", apiYamlData)
	prodAIRatelimit, sandAIRatelimit := prepareAIRatelimit(apiYamlData.MaxTps)
	endpointRes := getEndpointConfigs(sandboxURL, prodURL, endCertAvailable, endpointCertList, endpointSecurityData, apiUniqueID, prodAIRatelimit, sandAIRatelimit)

	apk.EndpointConfigurations = &endpointRes

	//Adding client-certificate configurations to the conf
	var certList CertDescriptor
	certAvailable := false

	if certArtifact.ClientCerts != "" {
		certErr := json.Unmarshal([]byte(certArtifact.ClientCerts), &certList)
		if certErr != nil {
			logger.LoggerTransformer.Errorf("Error while unmarshalling client_cert.json content: %v", apiYamlError)
			return "", "null", 0, nil, EndpointSecurityConfig{}, nil, nil, nil, certErr
		}
		certAvailable = true
	}

	authConfigList := mapAuthConfigs(apiYamlData.ID, apiYamlData.AuthorizationHeader, apiYamlData.APIKeyHeader,
		apiYamlData.SecuritySchemes, certAvailable, certList, apiUniqueID)

	apk.Authentication = &authConfigList

	corsEnabled := apiYamlData.CORSConfiguration.CORSConfigurationEnabled

	if corsEnabled {
		apk.CorsConfig = &apiYamlData.CORSConfiguration
	}

	aditionalProperties := make([]AdditionalProperty, len(apiYamlData.AdditionalProperties))

	for i, property := range apiYamlData.AdditionalProperties {
		prop := &AdditionalProperty{
			Name:  property.Name,
			Value: property.Value,
		}
		aditionalProperties[i] = *prop
	}

	apk.AdditionalProperties = &aditionalProperties

	c, marshalError := yaml.Marshal(apk)

	if marshalError != nil {
		logger.LoggerTransformer.Error("Error while marshalling apk yaml", marshalError)
		return "", "null", 0, nil, EndpointSecurityConfig{}, nil, prodAIRatelimit, sandAIRatelimit, marshalError
	}
	return string(c), apiYamlData.RevisionedAPIID, apiYamlData.RevisionID, configuredRateLimitPoliciesMap, endpointSecurityData, apk, prodAIRatelimit, sandAIRatelimit, nil
}

// Generate the interceptor policy if request or response policy exists
func getReqAndResInterceptors(reqPolicyCount, resPolicyCount int, reqPolicies []APIMOperationPolicy, resPolicies []APIMOperationPolicy) (*[]OperationPolicy, *[]OperationPolicy) {
	var requestPolicyList, responsePolicyList []OperationPolicy
	var interceptorParams *InterceptorService
	var requestInterceptorPolicy, responseInterceptorPolicy, requestBackendJWTPolicy OperationPolicy
	var mirrorRequestPolicy OperationPolicy
	var mirrorUrls []string

	if reqPolicyCount > 0 {
		for _, reqPolicy := range reqPolicies {
			logger.LoggerTransformer.Debugf("Request Policy: %v", reqPolicy)
			if reqPolicy.PolicyName == constants.InterceptorService {
				logger.LoggerTransformer.Debugf("Interceptor Type Request Policy: %v", reqPolicy)
				logger.LoggerTransformer.Debugf("Interceptor Service URL: %v", reqPolicy.Parameters[interceptorServiceURL])
				logger.LoggerTransformer.Debugf("Interceptor Includes: %v", reqPolicy.Parameters[includes])
				interceptorServiceURL := reqPolicy.Parameters[interceptorServiceURL].(string)
				includes := reqPolicy.Parameters[includes].(string)
				substrings := strings.Split(includes, ",")
				bodyEnabled := false
				headerEnabled := false
				trailersEnabled := false
				contextEnabled := false
				sslEnabled := false
				tlsSecretName := ""
				tlsSecretKey := ""
				for _, substring := range substrings {
					if strings.Contains(substring, requestHeader) {
						headerEnabled = true
					} else if strings.Contains(substring, requestBody) {
						bodyEnabled = true
					} else if strings.Contains(substring, requestTrailers) {
						trailersEnabled = true
					} else if strings.Contains(substring, requestContext) {
						contextEnabled = true
					}
				}

				if strings.Contains(interceptorServiceURL, https) {
					sslEnabled = true
				}

				if sslEnabled {
					tlsSecretName = reqPolicy.PolicyID + requestInterceptorSecretName
					tlsSecretKey = tlsKey
				}

				interceptorParams = &InterceptorService{
					BackendURL:      interceptorServiceURL,
					HeadersEnabled:  headerEnabled,
					BodyEnabled:     bodyEnabled,
					TrailersEnabled: trailersEnabled,
					ContextEnabled:  contextEnabled,
					TLSSecretName:   tlsSecretName,
					TLSSecretKey:    tlsSecretKey,
				}

				// Create an instance of OperationPolicy
				requestInterceptorPolicy = OperationPolicy{
					PolicyName:    interceptorPolicy,
					PolicyVersion: v1,
					Parameters:    interceptorParams,
				}
			} else if reqPolicy.PolicyName == constants.BackendJWT {
				encoding := reqPolicy.Parameters[encoding].(string)
				header := reqPolicy.Parameters[header].(string)
				signingAlgorithm := reqPolicy.Parameters[signingAlgorithm].(string)
				tokenTTL := reqPolicy.Parameters[tokenTTL].(string)
				tokenTTLConverted, err := strconv.Atoi(tokenTTL)
				if err != nil {
					logger.LoggerTransformer.Errorf("Error while converting tokenTTL to integer: %v", err)
				}

				if encoding == base64Url {
					encoding = base64url
				}

				backendJWTParams := &BackendJWT{
					Encoding:         encoding,
					Header:           header,
					SigningAlgorithm: signingAlgorithm,
					TokenTTL:         tokenTTLConverted,
				}

				// Create an instance of OperationPolicy
				requestBackendJWTPolicy = OperationPolicy{
					PolicyName:    backendJWTPolicy,
					PolicyVersion: v1,
					Parameters:    backendJWTParams,
				}
			} else if reqPolicy.PolicyName == constants.AddHeader {
				logger.LoggerTransformer.Debugf("AddHeader Type Request Policy: %v", reqPolicy)
				requestAddHeader := OperationPolicy{
					PolicyName:    addHeaderPolicy,
					PolicyVersion: v1,
					Parameters: Header{
						HeaderName:  reqPolicy.Parameters[headerName].(string),
						HeaderValue: reqPolicy.Parameters[headerValue].(string),
					},
				}
				requestPolicyList = append(requestPolicyList, requestAddHeader)
			} else if reqPolicy.PolicyName == constants.RemoveHeader {
				logger.LoggerTransformer.Debugf("RemoveHeader Type Request Policy: %v", reqPolicy)
				requestRemoveHeader := OperationPolicy{
					PolicyName:    removeHeaderPolicy,
					PolicyVersion: v1,
					Parameters: Header{
						HeaderName: reqPolicy.Parameters[headerName].(string),
					},
				}
				requestPolicyList = append(requestPolicyList, requestRemoveHeader)
			} else if reqPolicy.PolicyName == constants.RedirectRequest {
				logger.LoggerTransformer.Debugf("RedirectRequest Type Request Policy: %v", reqPolicy)
				redirectRequestPolicy := OperationPolicy{
					PolicyName:    requestRedirectPolicy,
					PolicyVersion: v1,
				}
				parameters := RedirectPolicy{
					URL: reqPolicy.Parameters[url].(string),
				}
				switch v := reqPolicy.Parameters[statusCode].(type) {
				case int:
					parameters.StatusCode = v
				case string:
					if intValue, err := strconv.Atoi(v); err == nil {
						parameters.StatusCode = intValue
					} else {
						logger.LoggerTransformer.Error("Invalid status code provided.")
					}
				default:
					parameters.StatusCode = 302
				}
				redirectRequestPolicy.Parameters = parameters
				requestPolicyList = append(requestPolicyList, redirectRequestPolicy)
			} else if reqPolicy.PolicyName == constants.MirrorRequest {
				logger.LoggerTransformer.Debugf("MirrorRequest Type Request Policy: %v", reqPolicy)
				if mirrorRequestPolicy.PolicyName == "" {
					mirrorRequestPolicy = OperationPolicy{
						PolicyName:    requestMirrorPolicy,
						PolicyVersion: v1,
					}
					mirrorUrls = []string{}
				}
				if reqPolicyParameters, ok := reqPolicy.Parameters[url]; ok {
					url := reqPolicyParameters.(string)
					mirrorUrls = append(mirrorUrls, url)
				}
			}
		}
	}

	if resPolicyCount > 0 {
		for _, resPolicy := range resPolicies {
			if resPolicy.PolicyName == constants.InterceptorService {
				interceptorServiceURL := resPolicy.Parameters[interceptorServiceURL].(string)
				includes := resPolicy.Parameters[includes].(string)
				substrings := strings.Split(includes, ",")
				bodyEnabled := false
				headerEnabled := false
				trailersEnabled := false
				contextEnabled := false
				sslEnabled := false
				tlsSecretName := ""
				tlsSecretKey := ""
				for _, substring := range substrings {
					if strings.Contains(substring, requestHeader) {
						headerEnabled = true
					} else if strings.Contains(substring, requestBody) {
						bodyEnabled = true
					} else if strings.Contains(substring, requestTrailers) {
						trailersEnabled = true
					} else if strings.Contains(substring, requestContext) {
						contextEnabled = true
					}
				}

				if strings.Contains(interceptorServiceURL, https) {
					sslEnabled = true
				}

				if sslEnabled {
					tlsSecretName = resPolicies[0].PolicyID + responseInterceptorSecretName
					tlsSecretKey = tlsKey
				}

				interceptorParams = &InterceptorService{
					BackendURL:      interceptorServiceURL,
					HeadersEnabled:  headerEnabled,
					BodyEnabled:     bodyEnabled,
					TrailersEnabled: trailersEnabled,
					ContextEnabled:  contextEnabled,
					TLSSecretName:   tlsSecretName,
					TLSSecretKey:    tlsSecretKey,
				}

				// Create an instance of OperationPolicy
				responseInterceptorPolicy = OperationPolicy{
					PolicyName:    interceptorPolicy,
					PolicyVersion: v1,
					Parameters:    interceptorParams,
				}
			} else if resPolicy.PolicyName == constants.AddHeader {
				logger.LoggerTransformer.Debugf("AddHeader Type Response Policy: %v", resPolicy)

				responseAddHeader := OperationPolicy{
					PolicyName:    addHeaderPolicy,
					PolicyVersion: v2,
					Parameters: Header{
						HeaderName:  resPolicy.Parameters[headerName].(string),
						HeaderValue: resPolicy.Parameters[headerValue].(string),
					},
				}
				responsePolicyList = append(responsePolicyList, responseAddHeader)
			} else if resPolicy.PolicyName == constants.RemoveHeader {
				logger.LoggerTransformer.Debugf("RemoveHeader Type Response Policy: %v", resPolicy)
				responseRemoveHeader := OperationPolicy{
					PolicyName:    removeHeaderPolicy,
					PolicyVersion: v1,
					Parameters: Header{
						HeaderName: resPolicy.Parameters[headerName].(string),
					},
				}
				responsePolicyList = append(responsePolicyList, responseRemoveHeader)
			}
		}
	}

	if reqPolicyCount > 0 {
		if requestInterceptorPolicy.PolicyName != "" {
			requestPolicyList = append(requestPolicyList, requestInterceptorPolicy)
		}
		if requestBackendJWTPolicy.PolicyName != "" {
			requestPolicyList = append(requestPolicyList, requestBackendJWTPolicy)
		}
		if mirrorRequestPolicy.PolicyName != "" {
			mirrorRequestPolicy.Parameters = URLList{
				URLs: mirrorUrls,
			}
			requestPolicyList = append(requestPolicyList, mirrorRequestPolicy)
		}
	}

	if resPolicyCount > 0 {
		if responseInterceptorPolicy.PolicyName != "" {
			responsePolicyList = append(responsePolicyList, responseInterceptorPolicy)
		}
	}
	return &requestPolicyList, &responsePolicyList
}

// prepareAIRatelimit Function that accepts apiYamlData and returns AIRatelimit
func prepareAIRatelimit(maxTps *MaxTps) (*AIRatelimit, *AIRatelimit) {
	if maxTps == nil {
		return nil, nil
	}
	prodAIRL := &AIRatelimit{}
	if maxTps.TokenBasedThrottlingConfiguration == nil ||
		maxTps.TokenBasedThrottlingConfiguration.IsTokenBasedThrottlingEnabled == nil ||
		maxTps.TokenBasedThrottlingConfiguration.ProductionMaxPromptTokenCount == nil ||
		maxTps.TokenBasedThrottlingConfiguration.ProductionMaxCompletionTokenCount == nil ||
		maxTps.TokenBasedThrottlingConfiguration.ProductionMaxTotalTokenCount == nil ||
		maxTps.ProductionTimeUnit == nil {
		prodAIRL = nil
	} else {
		prodAIRL = &AIRatelimit{
			Enabled: *maxTps.TokenBasedThrottlingConfiguration.IsTokenBasedThrottlingEnabled,
			Token: TokenAIRL{
				PromptLimit:     *maxTps.TokenBasedThrottlingConfiguration.ProductionMaxPromptTokenCount,
				CompletionLimit: *maxTps.TokenBasedThrottlingConfiguration.ProductionMaxCompletionTokenCount,
				TotalLimit:      *maxTps.TokenBasedThrottlingConfiguration.ProductionMaxTotalTokenCount,
				Unit:            CapitalizeFirstLetter(*maxTps.ProductionTimeUnit),
			},
			Request: RequestAIRL{
				RequestLimit: *maxTps.Production,
				Unit:         CapitalizeFirstLetter(*maxTps.ProductionTimeUnit),
			},
		}
	}
	sandAIRL := &AIRatelimit{}
	if maxTps.TokenBasedThrottlingConfiguration == nil ||
		maxTps.TokenBasedThrottlingConfiguration.IsTokenBasedThrottlingEnabled == nil ||
		maxTps.TokenBasedThrottlingConfiguration.SandboxMaxPromptTokenCount == nil ||
		maxTps.TokenBasedThrottlingConfiguration.SandboxMaxCompletionTokenCount == nil ||
		maxTps.TokenBasedThrottlingConfiguration.SandboxMaxTotalTokenCount == nil ||
		maxTps.SandboxTimeUnit == nil {
		sandAIRL = nil
	} else {
		sandAIRL = &AIRatelimit{
			Enabled: *maxTps.TokenBasedThrottlingConfiguration.IsTokenBasedThrottlingEnabled,
			Token: TokenAIRL{
				PromptLimit:     *maxTps.TokenBasedThrottlingConfiguration.SandboxMaxPromptTokenCount,
				CompletionLimit: *maxTps.TokenBasedThrottlingConfiguration.SandboxMaxCompletionTokenCount,
				TotalLimit:      *maxTps.TokenBasedThrottlingConfiguration.SandboxMaxTotalTokenCount,
				Unit:            CapitalizeFirstLetter(*maxTps.SandboxTimeUnit),
			},
			Request: RequestAIRL{
				RequestLimit: *maxTps.Sandbox,
				Unit:         CapitalizeFirstLetter(*maxTps.SandboxTimeUnit),
			},
		}
	}

	return prodAIRL, sandAIRL
}

// getEndpointConfigs will map the endpoints and there security configurations and returns them
// TODO: Currently the APK-Conf does not support giving multiple certs for a particular endpoint.
// After fixing this, the following logic should be changed to map multiple cert configs
func getEndpointConfigs(sandboxURL string, prodURL string, endCertAvailable bool, endpointCertList EndpointCertDescriptor, endpointSecurityData EndpointSecurityConfig, apiUniqueID string, prodAIRatelimit *AIRatelimit, sandAIRatelimit *AIRatelimit) EndpointConfigurations {
	var sandboxEndpointConf, prodEndpointConf EndpointConfiguration
	var sandBoxEndpointEnabled = false
	var prodEndpointEnabled = false
	if sandboxURL != "" {
		sandBoxEndpointEnabled = true
	}
	if prodURL != "" {
		prodEndpointEnabled = true
	}
	if prodAIRatelimit != nil {
		prodEndpointConf.AIRatelimit = *prodAIRatelimit
	}
	if sandAIRatelimit != nil {
		sandboxEndpointConf.AIRatelimit = *sandAIRatelimit
	}
	sandboxEndpointConf.Endpoint = sandboxURL
	prodEndpointConf.Endpoint = prodURL
	if endCertAvailable {
		for _, endCert := range endpointCertList.EndpointCertData {
			if endCert.Endpoint == sandboxURL {
				sandboxEndpointConf.EndCertificate = EndpointCertificate{
					Name: endCert.Alias,
					Key:  endCert.Certificate,
				}
			}
			if endCert.Endpoint == prodURL {
				prodEndpointConf.EndCertificate = EndpointCertificate{
					Name: endCert.Alias,
					Key:  endCert.Certificate,
				}
			}
		}
	}

	if endpointSecurityData.Sandbox.Enabled {
		sandboxEndpointConf.EndSecurity.Enabled = true
		if endpointSecurityData.Sandbox.Type == "apikey" {
			sandboxEndpointConf.EndSecurity.SecurityType = SecretInfo{
				SecretName:     strings.Join([]string{apiUniqueID, "sandbox", "secret"}, "-"),
				In:             "Header",
				APIKeyNameKey:  endpointSecurityData.Sandbox.APIKeyIdentifier,
				APIKeyValueKey: "apiKey",
			}
		} else {
			sandboxEndpointConf.EndSecurity.SecurityType = SecretInfo{
				SecretName:  strings.Join([]string{apiUniqueID, "sandbox", "secret"}, "-"),
				UsernameKey: "username",
				PasswordKey: "password",
			}
		}
	}

	if endpointSecurityData.Production.Enabled {
		prodEndpointConf.EndSecurity.Enabled = true
		if endpointSecurityData.Production.Type == "apikey" {
			prodEndpointConf.EndSecurity.SecurityType = SecretInfo{
				SecretName:     strings.Join([]string{apiUniqueID, "production", "secret"}, "-"),
				In:             "Header",
				APIKeyNameKey:  endpointSecurityData.Production.APIKeyIdentifier,
				APIKeyValueKey: "apiKey",
			}
		} else {
			prodEndpointConf.EndSecurity.SecurityType = SecretInfo{
				SecretName:  strings.Join([]string{apiUniqueID, "production", "secret"}, "-"),
				UsernameKey: "username",
				PasswordKey: "password",
			}
		}
	}

	epconfigs := EndpointConfigurations{}
	if sandBoxEndpointEnabled && prodEndpointEnabled {
		epconfigs = EndpointConfigurations{
			Sandbox:    &sandboxEndpointConf,
			Production: &prodEndpointConf,
		}
	} else if sandBoxEndpointEnabled {
		epconfigs = EndpointConfigurations{
			Sandbox: &sandboxEndpointConf,
		}
	} else if prodEndpointEnabled {
		epconfigs = EndpointConfigurations{
			Production: &prodEndpointConf,
		}
	}
	return epconfigs
}

// mapAuthConfigs will take the security schemes as the parameter and will return the mapped auth configs to be
// added into the apk-conf
func mapAuthConfigs(apiUUID string, authHeader string, configuredAPIKeyHeader string, securitySchemes []string, certAvailable bool, certList CertDescriptor, apiUniqueID string) []AuthConfiguration {
	var authConfigs []AuthConfiguration
	if StringExists(oAuth2SecScheme, securitySchemes) {
		var oauth2Config AuthConfiguration
		oauth2Config.AuthType = oAuth2
		oauth2Config.Enabled = true
		oauth2Config.HeaderName = authHeader
		if StringExists(applicationSecurityMandatory, securitySchemes) {
			oauth2Config.Required = mandatory
		} else {
			oauth2Config.Required = optional
		}

		authConfigs = append(authConfigs, oauth2Config)
	}
	if !StringExists("oauth2", securitySchemes) {
		oAuth2DisabledConfig := AuthConfiguration{
			AuthType: oAuth2,
			Enabled:  false,
		}
		authConfigs = append(authConfigs, oAuth2DisabledConfig)
	}
	if StringExists(mutualSSL, securitySchemes) && certAvailable {
		var mtlsConfig AuthConfiguration
		mtlsConfig.AuthType = mTLS
		mtlsConfig.Enabled = true
		if StringExists(mutualSSLMandatory, securitySchemes) {
			mtlsConfig.Required = mandatory
		} else {
			mtlsConfig.Required = optional
		}

		clientCerts := make([]Certificate, len(certList.CertData))

		for i, cert := range certList.CertData {
			prop := &Certificate{
				Name: apiUniqueID + "-" + cert.Alias,
				Key:  cert.Certificate,
			}
			clientCerts[i] = *prop
		}
		mtlsConfig.Certificates = clientCerts
		authConfigs = append(authConfigs, mtlsConfig)
	}

	internalKeyAuthConfig := AuthConfiguration{
		AuthType:   jwt,
		Enabled:    true,
		Audience:   []string{apiUUID},
		HeaderName: internalKeyHeader,
	}
	authConfigs = append(authConfigs, internalKeyAuthConfig)

	if StringExists(apiKeySecScheme, securitySchemes) {
		apiKeyAuthConfig := AuthConfiguration{
			AuthType:       apiKey,
			Enabled:        true,
			HeaderName:     configuredAPIKeyHeader,
			HeaderEnabled:  true,
			QueryParamName: apiKeyHeader,
		}
		if StringExists(applicationSecurityMandatory, securitySchemes) {
			apiKeyAuthConfig.Required = mandatory
		} else if StringExists(applicationSecurityOptional, securitySchemes) {
			apiKeyAuthConfig.Required = optional
		}
		authConfigs = append(authConfigs, apiKeyAuthConfig)
	}
	return authConfigs
}
