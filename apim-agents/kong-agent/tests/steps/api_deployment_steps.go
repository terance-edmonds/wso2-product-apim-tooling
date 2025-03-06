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

package steps

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/cucumber/godog"
	"github.com/wso2/product-apim-tooling/apim-agents/kong-agent/tests/utils"
	"github.com/wso2/product-apim-tooling/apim-agents/kong-agent/tests/utils/constants"
)

var OASURL string

// APIDeploymentSteps registers all step definitions for API deployment scenarios.
func APIDeploymentSteps(s *godog.ScenarioContext, ctx *utils.SharedContext) {

	// // Scenario 1: API creation and subscription steps
	s.Step(`^I use the Payload file "([^"]*)"$`, func(file string) error { return iUseThePayloadFile(ctx, file) })
	s.Step(`^I use the OAS URL "([^"]*)"$`, func(url string) error { return iUseTheOASURL(ctx, url) })
	s.Step(`^make the import API Creation request using OAS "([^"]*)"$`, func(method string) error {
		return makeImportAPICreationRequest(ctx, method)
	})
	s.Step(`^make the API Revision Deployment request$`, func() error { return makeAPIRevisionDeploymentRequest(ctx) })

	s.Step(`^make the Change Lifecycle request$`, func() error { return makeChangeLifecycleRequest(ctx) })

	s.Step(`^make the Application Creation request with the name "([^"]*)"$`, func(name string) error {
		return makeApplicationCreationRequest(ctx, name)
	})
	s.Step(`^I have a KeyManager$`, func() error { return iHaveAKeyManager(ctx) })
	s.Step(`^make the Generate Keys request$`, func() error { return makeGenerateKeysRequest(ctx) })
	s.Step(`^make the Subscription request$`, func() error { return makeSubscriptionRequest(ctx) })
	s.Step(`^I get "([^"]*)" oauth keys for application$`, func(env string) error {
		return getOAuthKeysForApplication(ctx, env)
	})
	s.Step(`^make the Access Token Generation request for "([^"]*)"$`, func(env string) error {
		return makeAccessTokenGenerationRequest(ctx, env)
	})

	// s.Step(`^I eventually receive (\d+) response code, not accepting$`, func(code int, table *godog.Table) error {
	// 	return iEventuallyReceiveResponseCodeNotAccepting(ctx, code, table)
	// })

	// // Scenario 2: API undeployment steps
	// s.Step(`^I delete the application "([^"]*)" from devportal$`, func(name string) error {
	// 	return iDeleteTheApplicationFromDevportal(ctx, name)
	// })
	// s.Step(`^I find the apiUUID of the API created with the name "([^"]*)"$`, func(name string) error {
	// 	return iFindTheApiUUIDOfTheAPICreatedWithTheName(ctx, name)
	// })
	// s.Step(`^I undeploy the selected API$`, func() error { return iUndeployTheSelectedAPI(ctx) })
}

// iHaveTheAPIPayloadFile loads the API payload file by its name.
func iUseThePayloadFile(ctx *utils.SharedContext, payloadFileName string) error {
	// Get the file path using the payload file name
	payloadFilePath := fmt.Sprintf("resources/%s", payloadFileName) // Adjust this to your file's actual location
	_, err := os.Stat(payloadFilePath)                              // Check if the file exists
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", payloadFilePath)
		}
		return err
	}

	// Store the file path in the context
	ctx.AddStoreValue("payloadFile", payloadFilePath)

	return nil
}

// iUseTheOASURL sets the OpenAPI Specification (OAS) URL for the test.
func iUseTheOASURL(ctx *utils.SharedContext, url string) error {
	ctx.AddStoreValue("OASURL", url)
	return nil
}

// makeImportAPICreationRequest handles API import request based on definition type (URL or File)
func makeImportAPICreationRequest(ctx *utils.SharedContext, definitionType string) error {
	var oasURL string
	var payloadFilePath string

	if url, ok := ctx.GetStoreValue("OASURL").(string); ok {
		oasURL = url
	}
	if filePath, ok := ctx.GetStoreValue("payloadFile").(string); ok {
		payloadFilePath = filePath
	}

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// If definition type is URL
	if definitionType == "URL" {
		fmt.Println("OAS URL: ", oasURL)
		_ = writer.WriteField("url", oasURL)
		utils.AddFileToMultipart(writer, "additionalProperties", payloadFilePath)
	}

	// If definition type is File
	if definitionType == "File" {
		fmt.Println("OAS File: ", payloadFilePath)
		utils.AddFileToMultipart(writer, "file", payloadFilePath)
		utils.AddFileToMultipart(writer, "additionalProperties", payloadFilePath)
	}

	// Close the multipart writer
	err := writer.Close()
	if err != nil {
		return fmt.Errorf("error closing multipart writer: %v", err)
	}

	// Prepare headers
	headers := map[string]string{
		"Authorization": "Bearer " + ctx.GetPublisherAccessToken(),
		"Host":          constants.DefaultAPIMAPIHost,
	}

	// Send the HTTP POST request
	req, err := http.NewRequest("POST", utils.GetImportAPIURL(), &requestBody)
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %v", err)
	}

	// Set the headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Set the content type for the multipart request
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Store the response and extract relevant data
	ctx.SetResponse(resp)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}

	// Store response body and extract API UUID
	ctx.SetResponseBody(string(body))
	apiUUID, err := utils.ExtractID(string(body))
	if err != nil {
		return fmt.Errorf("error extracting api UUID from body: %v", err)
	}
	ctx.SetApiUUID(apiUUID)

	// Simulate a sleep like in the Java version
	time.Sleep(3 * time.Second)

	return nil
}

// makeAPIRevisionDeploymentRequest makes a request to deploy an API revision.
func makeAPIRevisionDeploymentRequest(ctx *utils.SharedContext) error {
	httpclient := ctx.GetHTTPClient()
	apiUUID := ctx.GetApiUUID()
	fmt.Printf("API UUID: %s", apiUUID)

	// Prepare payload for revision creation
	payload := "{\"description\":\"Initial Revision\"}"
	headers := map[string]string{}
	headers[constants.RequestHeaders.Authorization] = "Bearer " + ctx.GetPublisherAccessToken()
	headers[constants.RequestHeaders.Host] = constants.DefaultAPIMAPIHost

	// Make the API revision request
	apiRevisionURL := utils.GetAPIRevisionURL(apiUUID)
	response, err := httpclient.DoPost(apiRevisionURL, headers, payload, constants.ContentTypes.ApplicationJSON)
	if err != nil {
		return fmt.Errorf("failed to create API revision: %v", err)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}

	// Extract and store revision UUID from the response
	revisionUUID, err := utils.ExtractID(string(body))
	if err != nil {
		return fmt.Errorf("error extracting revision UUID from body: %v", err)
	}
	ctx.SetRevisionUUID(revisionUUID)

	// Sleep for 3 seconds to simulate the delay
	time.Sleep(3 * time.Second)

	// Prepare payload for deployment
	payload2 := "[{\"name\": \"Default\", \"vhost\": \"default.gw.wso2.com\", \"displayOnDevportal\": true}]"

	// Make the API revision deployment request
	apiRevisionDeploymentURL := utils.GetAPIRevisionDeploymentURL(apiUUID, revisionUUID)
	response2, err := httpclient.DoPost(apiRevisionDeploymentURL, headers, payload2, constants.ContentTypes.ApplicationJSON)
	if err != nil {
		return fmt.Errorf("failed to deploy API revision: %v", err)
	}

	// Log the response and set it in the context
	fmt.Printf("Response: %+v", response2)
	ctx.SetResponse(response2)

	// Sleep for 3 seconds to simulate the delay
	time.Sleep(3 * time.Second)

	return nil
}

// makeChangeLifecycleRequest sends a request to change the lifecycle of an API.
func makeChangeLifecycleRequest(ctx *utils.SharedContext) error {
	httpclient := ctx.GetHTTPClient()
	apiUUID := ctx.GetApiUUID()
	payload := ""

	// Prepare headers
	headers := map[string]string{}
	headers[constants.RequestHeaders.Authorization] = "Bearer " + ctx.GetPublisherAccessToken()
	headers[constants.RequestHeaders.Host] = constants.DefaultAPIMAPIHost

	// Make the POST request
	url := utils.GetAPIChangeLifecycleURL(apiUUID)
	resp, err := httpclient.DoPost(url, headers, payload, constants.ContentTypes.ApplicationJSON)
	if err != nil {
		return err
	}

	// Store the response
	ctx.SetResponse(resp)

	// Wait for 3 seconds (simulating the sleep in the Java code)
	time.Sleep(3 * time.Second)

	return nil
}

// makeApplicationCreationRequest creates a new application in the system
func makeApplicationCreationRequest(ctx *utils.SharedContext, applicationName string) error {
	httpclient := ctx.GetHTTPClient()
	fmt.Printf("Creating an application\n")
	payload := fmt.Sprintf("{\"name\":\"%s\",\"throttlingPolicy\":\"10PerMin\",\"description\":\"test app\",\"tokenType\":\"JWT\",\"groups\":null,\"attributes\":{}}", applicationName)

	headers := map[string]string{}
	headers[constants.RequestHeaders.Authorization] = "Bearer " + ctx.GetDevportalAccessToken()
	headers[constants.RequestHeaders.Host] = constants.DefaultAPIMAPIHost

	response, err := httpclient.DoPost(utils.GetApplicationCreateURL(), headers, payload, constants.ContentTypes.ApplicationJSON)
	if err != nil {
		return err
	}

	ctx.SetResponse(response)
	responseBody, err := utils.ResponseEntityBodyToString(response)
	if err != nil {
		return err
	}
	ctx.SetResponseBody(responseBody)
	fmt.Printf("Response: %s\n", responseBody)

	applicationUUID, err := utils.ExtractApplicationID(responseBody)
	if err != nil {
		return err
	}
	ctx.SetApplicationUUID(applicationUUID)

	time.Sleep(3 * time.Second)

	return nil
}

// iHaveAKeyManager retrieves information about a key manager.
func iHaveAKeyManager(ctx *utils.SharedContext) error {
	httpclient := ctx.GetHTTPClient()
	headers := map[string]string{}
	headers[constants.RequestHeaders.Authorization] = "Bearer " + ctx.GetDevportalAccessToken()
	headers[constants.RequestHeaders.Host] = constants.DefaultAPIMAPIHost

	// Make HTTP GET request
	resp, err := httpclient.DoGet(utils.GetKeyManagerURL(), headers)
	if err != nil {
		return err
	}

	// Set response and extract the key manager UUID
	ctx.SetResponse(resp)
	body, err := utils.ResponseEntityBodyToString(resp)
	if err != nil {
		return err
	}
	ctx.SetResponseBody(body)

	keyManagerUUID, err := utils.ExtractKeyManagerID(body)
	if err != nil {
		return err
	}
	ctx.SetKeyManagerUUID(keyManagerUUID)

	// Wait for the response processing
	time.Sleep(3 * time.Second)

	return nil
}

// makeGenerateKeysRequest generates keys for the given application and key manager.
func makeGenerateKeysRequest(ctx *utils.SharedContext) error {
	httpclient := ctx.GetHTTPClient()
	applicationUUID := ctx.GetApplicationUUID()
	keyManagerUUID := ctx.GetKeyManagerUUID()
	fmt.Printf("Key Manager UUID: %s\n", keyManagerUUID)
	fmt.Printf("Application UUID: %s\n", applicationUUID)

	// Prepare payloads for production and sandbox keys
	payloadForProdKeys := fmt.Sprintf(`{
		"keyType":"PRODUCTION",
		"grantTypesToBeSupported":["password","client_credentials"],
		"callbackUrl":"",
		"additionalProperties":{
			"application_access_token_expiry_time":"N/A",
			"user_access_token_expiry_time":"N/A",
			"refresh_token_expiry_time":"N/A",
			"id_token_expiry_time":"N/A",
			"pkceMandatory":"false",
			"pkceSupportPlain":"false",
			"bypassClientCredentials":"false"
		},
		"keyManager":"%s",
		"validityTime":3600,
		"scopes":["default"]
	}`, keyManagerUUID)

	payloadForSandboxKeys := fmt.Sprintf(`{
		"keyType":"SANDBOX",
		"grantTypesToBeSupported":["password","client_credentials"],
		"callbackUrl":"",
		"additionalProperties":{
			"application_access_token_expiry_time":"N/A",
			"user_access_token_expiry_time":"N/A",
			"refresh_token_expiry_time":"N/A",
			"id_token_expiry_time":"N/A",
			"pkceMandatory":"false",
			"pkceSupportPlain":"false",
			"bypassClientCredentials":"false"
		},
		"keyManager":"%s",
		"validityTime":3600,
		"scopes":["default"]
	}`, keyManagerUUID)

	// Prepare headers
	headers := map[string]string{}
	headers[constants.RequestHeaders.Authorization] = "Bearer " + ctx.GetDevportalAccessToken()
	headers[constants.RequestHeaders.Host] = constants.DefaultAPIMAPIHost

	// Send request for production keys
	resp, err := httpclient.DoPost(utils.GetGenerateKeysURL(applicationUUID), headers, payloadForProdKeys, constants.ContentTypes.ApplicationJSON)
	if err != nil {
		return err
	}

	// Process response for production keys
	ctx.SetResponse(resp)
	body, err := utils.ResponseEntityBodyToString(resp)
	if err != nil {
		return err
	}
	ctx.SetResponseBody(body)

	prodConsumerSecret, err := utils.ExtractKeys(body, "consumerSecret")
	if err != nil {
		return err
	}
	ctx.SetConsumerSecret(prodConsumerSecret, "production")

	prodConsumerKey, err := utils.ExtractKeys(body, "consumerKey")
	if err != nil {
		return err
	}
	ctx.SetConsumerKey(prodConsumerKey, "production")

	prodKeyMappingId, err := utils.ExtractKeys(body, "keyMappingId")
	if err != nil {
		return err
	}
	ctx.SetKeyMappingID(prodKeyMappingId, "production")

	// Wait for response processing
	time.Sleep(3 * time.Second)

	// Send request for sandbox keys
	resp2, err := httpclient.DoPost(utils.GetGenerateKeysURL(applicationUUID), headers, payloadForSandboxKeys, constants.ContentTypes.ApplicationJSON)
	if err != nil {
		return err
	}

	// Process response for sandbox keys
	ctx.SetResponse(resp2)
	body2, err := utils.ResponseEntityBodyToString(resp2)
	if err != nil {
		return err
	}
	ctx.SetResponseBody(body2)

	sandConsumerSecret, err := utils.ExtractKeys(body2, "consumerSecret")
	if err != nil {
		return err
	}
	ctx.SetConsumerSecret(sandConsumerSecret, "sandbox")

	sandConsumerKey, err := utils.ExtractKeys(body2, "consumerKey")
	if err != nil {
		return err
	}
	ctx.SetConsumerKey(sandConsumerKey, "sandbox")

	sandKeyMappingId, err := utils.ExtractKeys(body2, "keyMappingId")
	if err != nil {
		return err
	}
	ctx.SetKeyMappingID(sandKeyMappingId, "sandbox")

	// Wait for response processing
	time.Sleep(3 * time.Second)

	return nil
}

// makeSubscriptionRequest makes a subscription request for the application and API.
func makeSubscriptionRequest(ctx *utils.SharedContext) error {
	httpclient := ctx.GetHTTPClient()
	applicationUUID := ctx.GetApplicationUUID()
	apiUUID := ctx.GetApiUUID()
	fmt.Printf("API UUID: %s\n", apiUUID)
	fmt.Printf("Application UUID: %s\n", applicationUUID)

	// Prepare the payload for the subscription request
	payload := fmt.Sprintf(`{
		"apiId":"%s",
		"applicationId":"%s",
		"throttlingPolicy":"Unlimited"
	}`, apiUUID, applicationUUID)

	// Prepare headers
	headers := map[string]string{}
	headers[constants.RequestHeaders.Authorization] = "Bearer " + ctx.GetDevportalAccessToken()
	headers[constants.RequestHeaders.Host] = constants.DefaultAPIMAPIHost

	// Send the subscription request
	resp, err := httpclient.DoPost(utils.GetSubscriptionURL(), headers, payload, constants.ContentTypes.ApplicationJSON)
	if err != nil {
		return err
	}

	// Process the response
	ctx.SetResponse(resp)
	body, err := utils.ResponseEntityBodyToString(resp)
	if err != nil {
		return err
	}
	ctx.SetResponseBody(body)

	subscriptionId, err := utils.ExtractKeys(body, "subscriptionId")
	if err != nil {
		return err
	}
	ctx.SetSubscriptionID(subscriptionId)

	// Log the extracted subscription ID
	fmt.Printf("Extracted subscription ID: %s\n", ctx.GetSubscriptionID())

	// Wait for response processing
	time.Sleep(3 * time.Second)

	return nil
}

// getOAuthKeysForApplication retrieves OAuth keys for an application based on the provided type ("production" or "sandbox").
func getOAuthKeysForApplication(ctx *utils.SharedContext, keyType string) error {
	httpclient := ctx.GetHTTPClient()
	applicationUUID := ctx.GetApplicationUUID()

	// Set the key type based on the input ("production" or "sandbox")
	if keyType != "production" {
		keyType = "sandbox"
	}

	// Prepare headers
	headers := map[string]string{}
	headers[constants.RequestHeaders.Authorization] = "Bearer " + ctx.GetDevportalAccessToken()
	headers[constants.RequestHeaders.Host] = constants.DefaultAPIMAPIHost
	headers[constants.RequestHeaders.ContentType] = constants.ContentTypes.ApplicationJSON

	// Make GET request
	resp, err := httpclient.DoGet(utils.GetOauthKeysURL(applicationUUID), headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Store response and extract OAuth key UUID
	ctx.SetResponse(resp)
	body, err := utils.ResponseEntityBodyToString(resp)
	if err != nil {
		return err
	}
	ctx.SetResponseBody(body)
	oauthKeyUUID, err := utils.ExtractOAuthMappingID(body, ctx.GetKeyMappingID(keyType))
	if err != nil {
		return err
	}
	ctx.SetOauthKeyUUID(oauthKeyUUID)

	// Wait for the response to settle
	time.Sleep(3 * time.Second)

	return nil
}

// makeAccessTokenGenerationRequest generates an access token for the application based on the provided key type ("production" or "sandbox").
func makeAccessTokenGenerationRequest(ctx *utils.SharedContext, keyType string) error {
	httpclient := ctx.GetHTTPClient()
	applicationUUID := ctx.GetApplicationUUID()
	oauthKeyUUID := ctx.GetOauthKeyUUID()

	// Determine the key type (either "production" or "sandbox")
	if keyType != "production" {
		keyType = "sandbox"
	}

	// Fetch consumer secret for the specified key type
	consumerSecret := ctx.GetConsumerSecret(keyType)

	// Log the values
	fmt.Printf("Generating keys for: %s", keyType)
	fmt.Printf("Application UUID: %s", applicationUUID)
	fmt.Printf("Oauth Key UUID: %s", oauthKeyUUID)

	// Prepare the payload for access token generation
	payload := fmt.Sprintf("{\"consumerSecret\":\"%s\",\"validityPeriod\":3600,\"revokeToken\":null,"+
		"\"scopes\":[\"write:pets\",\"read:pets\",\"query:hero\"],\"additionalProperties\":{\"id_token_expiry_time\":3600,"+
		"\"application_access_token_expiry_time\":3600,\"user_access_token_expiry_time\":3600,\"bypassClientCredentials\":false,"+
		"\"pkceMandatory\":false,\"pkceSupportPlain\":false,\"refresh_token_expiry_time\":86400}}", consumerSecret)

	// Set headers
	headers := map[string]string{}
	headers[constants.RequestHeaders.Authorization] = "Bearer " + ctx.GetDevportalAccessToken()
	headers[constants.RequestHeaders.Host] = constants.DefaultAPIMAPIHost

	// Make the POST request for access token generation
	resp, err := httpclient.DoPost(utils.GetAccessTokenGenerationURL(applicationUUID, oauthKeyUUID), headers, payload, constants.ContentTypes.ApplicationJSON)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Handle the response
	ctx.SetResponse(resp)
	body, err := utils.ResponseEntityBodyToString(resp)
	if err != nil {
		return err
	}
	ctx.SetResponseBody(body)

	// Extract access token and store it
	accessToken, err := utils.ExtractKeys(body, "accessToken")
	if err != nil {
		return err
	}
	ctx.SetApiAccessToken(accessToken)
	ctx.AddStoreValue("accessToken", accessToken)

	// Log the access token
	fmt.Printf("Access Token: %s", ctx.GetApiAccessToken())

	// Wait for the response to settle
	time.Sleep(3 * time.Second)

	return nil
}
